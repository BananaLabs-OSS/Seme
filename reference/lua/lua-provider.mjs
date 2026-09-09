import crypto from "node:crypto";

const schema = {
  i64Type: "00000000000000000000000000009010",
  boolType: "00000000000000000000000000009020",
  stringType: "00000000000000000000000000009040",
  bytesType: "00000000000000000000000000009041",
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  program: "00000000000000000000000000009015",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  call: "00000000000000000000000000009060",
  resultType: "00000000000000000000000000009042",
  resultOk: "00000000000000000000000000009043",
  resultError: "00000000000000000000000000009044",
  optionType: "0000000000000000000000000000a050",
  optionNone: "0000000000000000000000000000a051",
  optionSome: "0000000000000000000000000000a052",
  fixedArrayType: "000000000000000000000000000090f2",
  sliceType: "000000000000000000000000000090f8",
  mapType: "0000000000000000000000000000a040",
  recordType: "00000000000000000000000000009030",
  recordField: "00000000000000000000000000009031",
  fieldRead: "00000000000000000000000000009032",
  fixedArrayConstruct: "000000000000000000000000000090f3",
  collectionLength: "000000000000000000000000000090f9",
  indexRead: "000000000000000000000000000090f4",
  integerLiteral: "00000000000000000000000000009070",
  emptyMap: "0000000000000000000000000000a041",
  mapLookup: "0000000000000000000000000000a042",
  mapUpdate: "0000000000000000000000000000a043",
};

/**
 * Lift the deliberately bounded Lua 5.1 / LuaJIT profile directly to Seme.
 * `sources` is an ordered array of { name, source }; file names never affect
 * semantic identity.
 */
export function liftLua({ sources, packagePath, revision, moduleG1, entryName }) {
  if (!Array.isArray(sources) || sources.length === 0) fail("lua.requires_sources");
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("lua.invalid_snapshot");
  const records = parseRecords(sources, packagePath);
  const declarations = sources.flatMap(({ name, source }) => parseSource(source, name, records));
  const byName = new Map();
  for (const declaration of declarations) {
    if (byName.has(declaration.name)) fail("lua.duplicate_function", declaration.location);
    byName.set(declaration.name, declaration);
  }
  const entry = entryName ? byName.get(entryName) : declarations.find((item) => item.exported);
  if (!entry) fail("lua.entry_not_found");
  if (!entry.exported) fail("lua.entry_must_be_global", entry.location);

  const additions = [];
  for (const type of declarations.flatMap((item) => [...item.parameters.map((parameter) => parameter.type), item.resultType])) ensureType(type, additions, records);
  const descriptions = declarations.map((declaration) => ({
    ...declaration,
    id: stableID("session-declaration", packagePath, declaration.name),
  }));
  const descriptionsByName = new Map(descriptions.map((item) => [item.name, item]));

  for (const description of descriptions) {
    const parameterIDs = description.parameters.map((parameter, index) => {
      const id = stableID("execution", description.id, "parameter", String(index));
      additions.push(graphEntity(id, entity(id, schema.parameter, [
        [0x9120, bytes(parameter.name)], [0x9121, ref(typeID(parameter.type))], [0x9122, `uu ${index}`],
      ])));
      return id;
    });
    const context = { description, parameterIDs, descriptionsByName, additions, records };
    const expression = emitExpression(description.expression, context, "body.statement.expression");
    const returnedID = stableID("execution", description.id, "body.statement", "return");
    const blockID = stableID("execution", description.id, "body", "block");
    additions.push(graphEntity(returnedID, entity(returnedID, schema.returned, [[0x9810, refs([expression])]])));
    additions.push(graphEntity(blockID, entity(blockID, schema.block, [[0x9800, refs([returnedID])]])));
    additions.push(graphEntity(description.id, entity(description.id, schema.function, [
      [0x9110, bytes(description.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(typeID(description.resultType))], [0x9113, ref(blockID)],
    ])));
  }
  const functionIDs = descriptions.map((item) => item.id).sort();
  const programID = stableID("session-program", packagePath);
  additions.push(graphEntity(programID, entity(programID, schema.program, [
    [0x9150, refs(functionIDs)], [0x9151, ref(stableID("session-declaration", packagePath, entry.name))],
  ])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), additions);
}

function parseSource(source, file, records) {
  if (typeof source !== "string" || typeof file !== "string" || !file) fail("lua.invalid_source");
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  const declarations = [];
  let annotations = [];
  for (let index = 0; index < lines.length;) {
    const trimmed = lines[index].trim();
    if (!trimmed || trimmed.startsWith("--") && !trimmed.startsWith("---@")) { index += 1; continue; }
    if (trimmed.startsWith("---@class ")) { index += 1; while (index < lines.length && lines[index].trim().startsWith("---@field ")) index += 1; continue; }
    if (trimmed.startsWith("---@")) { annotations.push({ text: trimmed, line: index + 1 }); index += 1; continue; }
    const match = /^(local\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*$/.exec(trimmed);
    if (!match) fail("lua.unsupported_top_level", { file, line: index + 1, column: 1 });
    const location = { file, line: index + 1, column: lines[index].indexOf("function") + 1 };
    const parameters = match[3].trim() ? match[3].split(",").map((item) => item.trim()) : [];
    if (parameters.some((item) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(item))) fail("lua.unsupported_parameter", location);
    const signature = validateAnnotations(annotations, parameters, location, records);
    annotations = [];
    index += 1;
    const body = [];
    while (index < lines.length && lines[index].trim() !== "end") { body.push({ text: lines[index], line: index + 1 }); index += 1; }
    if (index >= lines.length) fail("lua.unclosed_function", location);
    if (body.filter((item) => item.text.trim()).length !== 1) fail("lua.function_body_profile", location);
    const statement = body.find((item) => item.text.trim());
    const returned = /^\s*return\s+(.+?)\s*$/.exec(statement.text);
    if (!returned) fail("lua.requires_return", { file, line: statement.line, column: 1 });
    declarations.push({ name: match[2], parameters: signature.parameters, resultType: signature.resultType, expression: parseExpression(returned[1], file, statement.line), exported: !match[1], location });
    index += 1;
  }
  if (annotations.length) fail("lua.orphan_annotation", { file, line: annotations[0].line, column: 1 });
  return declarations;
}

function validateAnnotations(annotations, parameters, location, records) {
  const declared = annotations.filter((item) => item.text.startsWith("---@param ")).map((item) => /^---@param\s+([A-Za-z_][A-Za-z0-9_]*)\s+(\S+)$/.exec(item.text));
  const result = annotations.filter((item) => item.text.startsWith("---@return "));
  const returned = result.length === 1 ? /^---@return\s+(\S+)$/.exec(result[0].text) : null;
  if (declared.some((item) => !item) || !returned) fail("lua.unsupported_annotation", location);
  if (declared.length !== parameters.length || declared.some((item, index) => item[1] !== parameters[index])) fail("lua.signature_mismatch", location);
  if (annotations.length !== declared.length + 1) fail("lua.unsupported_annotation", location);
  return { parameters: declared.map((item) => ({ name: item[1], type: annotationType(item[2], location, records) })), resultType: annotationType(returned[1], location, records) };
}
function annotationType(value, location, records) {
  const scalar = ({ boolean: "bool", "seme.i64": "i64", "seme.text": "text", "seme.bytes": "bytes" })[value];
  if (scalar) return scalar;
  const option = /^seme\.option<(.+)>$/.exec(value);
  if (option) return `option:${annotationType(option[1], location, records)}`;
  const generic = /^(seme\.(?:result|array|slice|map))<(.+)>$/.exec(value);
  if (generic) {
    const arguments_ = splitGenericArguments(generic[2], location);
    if (generic[1] === "seme.result" && arguments_.length === 2) return `result:${annotationType(arguments_[0], location, records)}:${annotationType(arguments_[1], location, records)}`;
    if (generic[1] === "seme.array" && arguments_.length === 2 && /^[1-9][0-9]*$/.test(arguments_[1])) return `array:${annotationType(arguments_[0], location, records)}:${arguments_[1]}`;
    if (generic[1] === "seme.slice" && arguments_.length === 1) return `slice:${annotationType(arguments_[0], location, records)}`;
    if (generic[1] === "seme.map" && arguments_.length === 2) return `map:${annotationType(arguments_[0], location, records)}:${annotationType(arguments_[1], location, records)}`;
  }
  if (records.has(value)) return `record:${records.get(value).id}:${value}`;
  fail("lua.unsupported_annotation", location);
}
function splitGenericArguments(value, location) {
  const result = []; let depth = 0; let start = 0;
  for (let index = 0; index < value.length; index += 1) {
    if (value[index] === "<") depth += 1;
    else if (value[index] === ">") { depth -= 1; if (depth < 0) fail("lua.unsupported_annotation", location); }
    else if (value[index] === "," && depth === 0) { result.push(value.slice(start, index)); start = index + 1; }
  }
  if (depth !== 0) fail("lua.unsupported_annotation", location);
  result.push(value.slice(start));
  return result;
}

function parseExpression(text, file, line) {
  const field = /^Seme\.field\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*"([A-Za-z_][A-Za-z0-9_]*)"\s*\)$/.exec(text);
  if (field) return { kind: "field", base: field[1], field: field[2], location: { file, line, column: 1 } };
  const array = /^Seme\.array\s*\((.*)\)$/.exec(text);
  if (array) return { kind: "array", arguments: array[1].trim() ? array[1].split(",").map((name) => name.trim()) : [], location: { file, line, column: 1 } };
  const length = /^Seme\.length\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$/.exec(text);
  if (length) return { kind: "length", base: length[1], location: { file, line, column: 1 } };
  const index = /^Seme\.index_zero\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([0-9]+)\s*\)$/.exec(text);
  if (index) return { kind: "index", base: index[1], index: index[2], location: { file, line, column: 1 } };
  const emptyMap = /^Seme\.empty_map\s*\(\s*"(i64|text|bytes)"\s*\)$/.exec(text);
  if (emptyMap) return { kind: "empty_map", descriptor: emptyMap[1], location: { file, line, column: 1 } };
  const lookup = /^Seme\.lookup_zero\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*"(i64|text|bytes)"\s*\)$/.exec(text);
  if (lookup) return { kind: "lookup_zero", base: lookup[1], key: lookup[2], descriptor: lookup[3], location: { file, line, column: 1 } };
  const mapOperation = /^Seme\.(map_update)\s*\(\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*,\s*([A-Za-z_][A-Za-z0-9_]*)\s*\)$/.exec(text);
  if (mapOperation) return { kind: mapOperation[1], base: mapOperation[2], key: mapOperation[3], value: mapOperation[4], location: { file, line, column: 1 } };
  const identifier = /^([A-Za-z_][A-Za-z0-9_]*)$/.exec(text);
  if (identifier) return { kind: "identifier", name: identifier[1], location: { file, line, column: 1 } };
  const constructor = /^Seme\.(some|none|ok|err)\s*\((.*)\)$/.exec(text);
  if (constructor) {
    const arguments_ = constructor[2].trim() ? [parseExpression(constructor[2].trim(), file, line)] : [];
    return { kind: constructor[1], arguments: arguments_, location: { file, line, column: 1 } };
  }
  const call = /^([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$/.exec(text);
  if (call) {
    const arguments_ = call[2].trim() ? call[2].split(",").map((item) => parseExpression(item.trim(), file, line)) : [];
    return { kind: "call", name: call[1], arguments: arguments_, location: { file, line, column: 1 } };
  }
  fail("lua.unsupported_expression", { file, line, column: 1 });
}

function emitExpression(expression, context, path) {
  if (expression.kind === "identifier") {
    const index = context.description.parameters.findIndex((parameter) => parameter.name === expression.name);
    if (index < 0) fail("lua.unknown_identifier", expression.location);
    if (context.description.parameters[index].type !== context.description.resultType) fail("lua.return_type", expression.location);
    const id = stableID("execution", context.description.id, "expression", path, "parameter-read");
    context.additions.push(graphEntity(id, entity(id, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
    return id;
  }
  if (["some", "none", "ok", "err"].includes(expression.kind)) return emitConstructor(expression, context, path);
  if (expression.kind === "field") {
    const index = context.description.parameters.findIndex((parameter) => parameter.name === expression.base);
    if (index < 0) fail("lua.unknown_identifier", expression.location);
    const parameterType = context.description.parameters[index].type;
    if (!parameterType.startsWith("record:")) fail("lua.field_receiver_type", expression.location);
    const record = context.records.get(parameterType.split(":").slice(2).join(":"));
    const fieldIndex = record?.fields.findIndex((field) => field.name === expression.field) ?? -1;
    if (fieldIndex < 0) fail("lua.unknown_field", expression.location);
    if (record.fields[fieldIndex].type !== context.description.resultType) fail("lua.field_result_type", expression.location);
    const readID = stableID("execution", context.description.id, "expression", `${path}.base`, "parameter-read");
    const id = stableID("execution", context.description.id, "expression", path, "field-read");
    context.additions.push(graphEntity(readID, entity(readID, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
    const fieldID = stableID("execution", record.id, "field", String(fieldIndex));
    context.additions.push(graphEntity(id, entity(id, schema.fieldRead, [[0x9320, ref(readID)], [0x9321, ref(fieldID)]])));
    return id;
  }
  if (["array", "length", "index", "empty_map", "lookup_zero", "map_update"].includes(expression.kind)) return emitCollectionExpression(expression, context, path);
  const callee = context.descriptionsByName.get(expression.name);
  if (!callee) fail("lua.unknown_call", expression.location);
  if (callee.parameters.length !== expression.arguments.length) fail("lua.call_arity", expression.location);
  if (callee.resultType !== context.description.resultType) fail("lua.call_result_type", expression.location);
  for (let index = 0; index < expression.arguments.length; index += 1) {
    const argument = expression.arguments[index];
    if (argument.kind !== "identifier") fail("lua.call_argument_profile", argument.location);
    const source = context.description.parameters.find((parameter) => parameter.name === argument.name);
    if (!source || source.type !== callee.parameters[index].type) fail("lua.call_argument_type", argument.location);
  }
  const arguments_ = expression.arguments.map((item, index) => emitExpression(item, context, `${path}.argument.${index}`));
  const id = stableID("execution", context.description.id, "expression", path, "call");
  context.additions.push(graphEntity(id, entity(id, schema.call, [[0x9600, ref(callee.id)], [0x9601, refs(arguments_)]])));
  return id;
}

function parameterRead(name, expected, context, path, location) {
  const index = context.description.parameters.findIndex((parameter) => parameter.name === name);
  if (index < 0) fail("lua.unknown_identifier", location);
  if (context.description.parameters[index].type !== expected) fail("lua.collection_argument_type", location);
  const id = stableID("execution", context.description.id, "expression", path, "parameter-read");
  context.additions.push(graphEntity(id, entity(id, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
  return id;
}
function emitCollectionExpression(expression, context, path) {
  const result = context.description.resultType;
  const id = stableID("execution", context.description.id, "expression", path, expression.kind);
  if (expression.kind === "array") {
    if (!result.startsWith("array:")) fail("lua.array_result_type", expression.location);
    const [, element, length] = result.split(":");
    if (expression.arguments.length !== Number(length)) fail("lua.array_arity", expression.location);
    const values = expression.arguments.map((name, index) => parameterRead(name, element, context, `${path}.element.${index}`, expression.location));
    context.additions.push(graphEntity(id, entity(id, schema.fixedArrayConstruct, [[0x9f30, ref(typeID(result))], [0x9f31, refs(values)]])));
    return id;
  }
  if (expression.kind === "empty_map") {
    if (!result.startsWith("map:")) fail("lua.map_result_type", expression.location);
    if (result.split(":")[2] !== expression.descriptor) fail("lua.map_descriptor_type", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.emptyMap, [[0xa0410, ref(typeID(result))]])));
    return id;
  }
  if (expression.kind === "length") {
    if (result !== "i64") fail("lua.length_result_type", expression.location);
    const parameter = context.description.parameters.find((item) => item.name === expression.base);
    if (!parameter || !(parameter.type.startsWith("array:") || parameter.type.startsWith("slice:"))) fail("lua.length_collection_type", expression.location);
    const base = parameterRead(expression.base, parameter.type, context, `${path}.collection`, expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.collectionLength, [[0x9f90, ref(base)]])));
    return id;
  }
  if (expression.kind === "index") {
    const parameter = context.description.parameters.find((item) => item.name === expression.base);
    if (!parameter || !parameter.type.startsWith("array:")) fail("lua.index_collection_type", expression.location);
    const [, element, length] = parameter.type.split(":");
    if (result !== element || BigInt(expression.index) >= BigInt(length)) fail("lua.index_type_or_range", expression.location);
    const base = parameterRead(expression.base, parameter.type, context, `${path}.collection`, expression.location);
    const literal = stableID("execution", context.description.id, "expression", `${path}.index`, "integer-literal");
    context.additions.push(graphEntity(literal, entity(literal, schema.integerLiteral, [[0x9700, `uu ${expression.index}`], [0x9701, ref(ids.i64)]])));
    context.additions.push(graphEntity(id, entity(id, schema.indexRead, [[0x9f40, ref(base)], [0x9f41, ref(literal)]])));
    return id;
  }
  const mapParameter = context.description.parameters.find((item) => item.name === expression.base);
  if (!mapParameter || !mapParameter.type.startsWith("map:")) fail("lua.map_receiver_type", expression.location);
  const [, keyType, valueType] = mapParameter.type.split(":");
  const map = parameterRead(expression.base, mapParameter.type, context, `${path}.map`, expression.location);
  const key = parameterRead(expression.key, keyType, context, `${path}.key`, expression.location);
  if (expression.kind === "lookup_zero") {
    if (result !== valueType || expression.value || expression.descriptor !== valueType) fail("lua.map_lookup_type", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.mapLookup, [[0xa0420, ref(map)], [0xa0421, ref(key)]])));
  } else {
    if (result !== mapParameter.type || !expression.value) fail("lua.map_update_type", expression.location);
    const value = parameterRead(expression.value, valueType, context, `${path}.value`, expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.mapUpdate, [[0xa0430, ref(map)], [0xa0431, ref(key)], [0xa0432, ref(value)]])));
  }
  return id;
}

function emitConstructor(expression, context, path) {
  const type = context.description.resultType;
  const isOption = type.startsWith("option:");
  const isResult = type.startsWith("result:");
  if ((expression.kind === "some" || expression.kind === "none") !== isOption || (expression.kind === "ok" || expression.kind === "err") !== isResult) fail("lua.constructor_result_type", expression.location);
  const id = stableID("execution", context.description.id, "expression", path, expression.kind);
  if (expression.kind === "none") {
    if (expression.arguments.length !== 0) fail("lua.constructor_arity", expression.location);
    context.additions.push(graphEntity(id, entity(id, schema.optionNone, [[0xa0510, ref(typeID(type))]])));
    return id;
  }
  if (expression.arguments.length !== 1 || expression.arguments[0].kind !== "identifier") fail("lua.constructor_arity", expression.location);
  const argument = expression.arguments[0];
  const index = context.description.parameters.findIndex((parameter) => parameter.name === argument.name);
  if (index < 0) fail("lua.unknown_identifier", argument.location);
  const expected = expression.kind === "some" ? type.slice("option:".length) : splitResult(type)[expression.kind === "ok" ? 0 : 1];
  if (context.description.parameters[index].type !== expected) fail("lua.constructor_value_type", argument.location);
  const readID = stableID("execution", context.description.id, "expression", `${path}.value`, "parameter-read");
  context.additions.push(graphEntity(readID, entity(readID, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
  const constructorSchema = expression.kind === "some" ? schema.optionSome : expression.kind === "ok" ? schema.resultOk : schema.resultError;
  const fields = expression.kind === "some" ? [[0xa0520, ref(typeID(type))], [0xa0521, ref(readID)]] : expression.kind === "ok" ? [[0x9410, ref(typeID(type))], [0x9411, ref(readID)]] : [[0x9420, ref(typeID(type))], [0x9421, ref(readID)]];
  context.additions.push(graphEntity(id, entity(id, constructorSchema, fields)));
  return id;
}

function splitResult(type) {
  const parts = type.slice("result:".length).split(":");
  if (parts.length !== 2) fail("lua.nested_result_profile");
  return parts;
}
function typeID(type) {
  if (ids[type]) return ids[type];
  if (type.startsWith("option:")) return stableID("execution", "type", "option", typeID(type.slice(7)));
  if (type.startsWith("result:")) { const [ok, error] = splitResult(type); return stableID("execution", "type", "result", typeID(ok), typeID(error)); }
  if (type.startsWith("array:")) { const [, element, length] = type.split(":"); return stableID("execution", "type", "fixed-array", element, length); }
  if (type.startsWith("slice:")) return stableID("execution", "type", "slice", type.slice(6));
  if (type.startsWith("map:")) { const [, key, value] = type.split(":"); return stableID("execution", "type", "map", key, value); }
  if (type.startsWith("record:")) return type.split(":")[1];
  fail("lua.unknown_type");
}
function ensureType(type, additions, records) {
  const add = (id, schemaID, fields) => { if (!additions.some((item) => item.id === id)) additions.push(graphEntity(id, entity(id, schemaID, fields))); };
  if (type === "i64") return add(ids.i64, schema.i64Type, [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]]);
  if (type === "bool") return add(ids.bool, schema.boolType, []);
  if (type === "text") return add(ids.text, schema.stringType, []);
  if (type === "bytes") return add(ids.bytes, schema.bytesType, []);
  if (type.startsWith("option:")) { const value = type.slice(7); ensureType(value, additions, records); return add(typeID(type), schema.optionType, [[0xa0500, ref(typeID(value))]]); }
  if (type.startsWith("result:")) { const [ok, error] = splitResult(type); ensureType(ok, additions, records); ensureType(error, additions, records); return add(typeID(type), schema.resultType, [[0x9400, ref(typeID(ok))], [0x9401, ref(typeID(error))]]); }
  if (type.startsWith("array:")) { const [, element, length] = type.split(":"); ensureType(element, additions, records); return add(typeID(type), schema.fixedArrayType, [[0x9f20, ref(typeID(element))], [0x9f21, `uu ${length}`]]); }
  if (type.startsWith("slice:")) { const element = type.slice(6); ensureType(element, additions, records); return add(typeID(type), schema.sliceType, [[0x9f80, ref(typeID(element))]]); }
  if (type.startsWith("map:")) { const [, key, value] = type.split(":"); ensureType(key, additions, records); ensureType(value, additions, records); return add(typeID(type), schema.mapType, [[0xa0400, ref(typeID(key))], [0xa0401, ref(typeID(value))]]); }
  if (type.startsWith("record:")) {
    const record = records.get(type.split(":").slice(2).join(":")); if (!record) fail("lua.unknown_record");
    const fieldIDs = record.fields.map((field, index) => { ensureType(field.type, additions, records); const id = stableID("execution", record.id, "field", String(index)); add(id, schema.recordField, [[0x9310, bytes(field.name)], [0x9311, ref(typeID(field.type))], [0x9312, `uu ${index}`]]); return id; });
    return add(record.id, schema.recordType, [[0x9300, bytes(record.name)], [0x9301, refs(fieldIDs)]]);
  }
  fail("lua.unknown_type");
}

function parseRecords(sources, packagePath) {
  const records = new Map();
  for (const { name: file, source } of sources) {
    const lines = source.replace(/\r\n?/g, "\n").split("\n");
    for (let index = 0; index < lines.length; index += 1) {
      const match = /^---@class\s+([A-Za-z_][A-Za-z0-9_]*)$/.exec(lines[index].trim());
      if (!match) continue;
      if (records.has(match[1])) fail("lua.duplicate_record", { file, line: index + 1, column: 1 });
      const record = { name: match[1], id: stableID("execution", "record", packagePath, match[1]), fields: [] };
      while (index + 1 < lines.length) {
        const field = /^---@field\s+([A-Za-z_][A-Za-z0-9_]*)\s+(\S+)$/.exec(lines[index + 1].trim());
        if (!field) break;
        record.fields.push({ name: field[1], sourceType: field[2], location: { file, line: index + 2, column: 1 } }); index += 1;
      }
      if (!record.fields.length || new Set(record.fields.map((field) => field.name)).size !== record.fields.length) fail("lua.invalid_record", { file, line: index + 1, column: 1 });
      records.set(record.name, record);
    }
  }
  for (const record of records.values()) record.fields = record.fields.map((field) => ({ ...field, type: annotationType(field.sourceType, field.location, records) }));
  return records;
}

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  text: stableID("execution", "type", "string"),
  bytes: stableID("execution", "type", "bytes"),
};
function stableID(...parts) { const hash = crypto.createHash("sha256").update("seme.provider.identity.v1\0"); for (const part of parts) hash.update(part).update("\0"); return `80${hash.digest().subarray(0, 15).toString("hex")}`; }
function graphEntity(id, text) { return { id, text }; }
function bytes(value) { return `by ${Buffer.from(value, "utf8").toString("hex") || "-"}`; }
function ref(value) { return `rf ${value}`; }
function refs(values) { return `li ${values.length}${values.map((value) => `\nrf ${value}`).join("")}`; }
function entity(id, type, fields) { fields.sort((a, b) => a[0] - b[0]); return `en ${id} ${type} 1 ${fields.length}\n${fields.map(([field, value]) => `fi ${field.toString(16).padStart(32, "0")} ${value}\n`).join("")}`; }
function compose(moduleG1, revision, additions) {
  if (typeof moduleG1 !== "string") fail("lua.requires_module");
  const entities = []; let current;
  for (const line of moduleG1.trim().split("\n")) { if (line.startsWith("en ")) { current = { id: line.split(/\s+/)[1], text: "" }; entities.push(current); } if (current) current.text += `${line}\n`; }
  const root = entities.find((item) => item.id === "00000000000000000000000000009000");
  if (Number(root?.text.match(/^en\s+\S+\s+\S+\s+(\d+)\s+/m)?.[1] ?? 0) >= 26) entities.length = 0;
  const unique = new Map(entities.map((item) => [item.id, item]));
  for (const item of additions) { const old = unique.get(item.id); if (old && old.text !== item.text) fail("lua.identity_collision"); unique.set(item.id, item); }
  const sorted = [...unique.values()].sort((a, b) => a.id.localeCompare(b.id));
  return `# Generated exact bounded Lua to Core Execution lift.\nve 1\nmo 00000000000000000000000000009000\nrv ${revision}\npc 0\nec ${sorted.length}\n${sorted.map((item) => `\n${item.text}`).join("")}`;
}
function fail(code, location) { throw new Error(`${code}${location ? `:${location.file}:${location.line}:${location.column}` : ""}`); }
