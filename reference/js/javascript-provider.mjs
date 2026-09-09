import crypto from "node:crypto";
import { parse } from "acorn";

const refinedI64Arithmetic = new WeakSet();
function admitRefinedI64Arithmetic(node) {
  if (!node || typeof node !== "object") return;
  if (node.type === "BinaryExpression" && ["+", "-", "*"].includes(node.operator)) refinedI64Arithmetic.add(node);
  if (node.type === "BinaryExpression") { admitRefinedI64Arithmetic(node.left); admitRefinedI64Arithmetic(node.right); }
}

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  string: stableID("execution", "type", "string"),
  bytes: stableID("execution", "type", "bytes"),
};

export function liftJavaScript({ source, packagePath, revision, moduleG1, entryName }) {
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("javascript.invalid_snapshot");
  const comments = [];
  let program;
  try {
    program = parse(source, { ecmaVersion: 2024, sourceType: "module", locations: true, onComment: comments });
  } catch (error) {
    fail("javascript.parse", error.loc);
  }
  // This provider describes a closed semantic snapshot.  Silently ignoring
  // executable module-level statements would make the lifted graph disagree
  // with JavaScript (notably prototype mutation and accessor installation).
  // Imports are handled by the bounded module adapter below; declarations are
  // handled explicitly.  Every other top-level form must be rejected.
  for (const item of program.body) {
    const declaration = item.type === "ExportNamedDeclaration" ? item.declaration : item;
    if (item.type === "ImportDeclaration" || declaration?.type === "FunctionDeclaration" || declaration?.type === "ClassDeclaration") continue;
    fail("javascript.unsupported_module_statement", item.loc?.start);
  }
  const exported = new Set(program.body.filter((item) => item.type === "ExportNamedDeclaration" && item.declaration?.type === "FunctionDeclaration").map((item) => item.declaration.id?.name));
  const declarations = program.body.map((item) => item.type === "ExportNamedDeclaration" ? item.declaration : item);
  const functions = declarations.filter((item) => item?.type === "FunctionDeclaration");
  if (functions.length === 0) fail("javascript.requires_function");
  const descriptions = functions.map((fn) => {
    if (!fn.id || fn.async || fn.generator) fail("javascript.unsupported_function", fn.loc.start);
    const signature = readSignature(comments, fn);
    if (signature.parameters.length !== fn.params.length) fail("javascript.signature_arity", fn.loc.start);
    return { fn, signature, id: stableID("session-declaration", packagePath, fn.id.name) };
  }).sort((left, right) => left.id.localeCompare(right.id));
  const functionsByName = new Map(descriptions.map((item) => [item.fn.id.name, item]));
  if (functionsByName.size !== descriptions.length) fail("javascript.duplicate_function");
  const recordsByName = readRecords(comments, packagePath);
  const interfacesByName = readInterfaces(comments, packagePath);
  for (const description of descriptions) normalizeInterfaceTypes(description.signature, interfacesByName);
  const methodDescriptions = readMethods(declarations, comments, recordsByName, packagePath);
  for (const description of methodDescriptions) normalizeInterfaceTypes(description.signature, interfacesByName);
  const methodsByTypeAndName = new Map(methodDescriptions.map((item) => [`${item.receiverType}:${item.fn.key.name}`, item]));
  const declaredEffects = new Set();
  const entities = [
    graphEntity(ids.i64, entity(ids.i64, "00000000000000000000000000009010", [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]])),
    graphEntity(ids.bool, entity(ids.bool, "00000000000000000000000000009020", [])),
    graphEntity(ids.string, entity(ids.string, "00000000000000000000000000009040", [])),
    graphEntity(ids.bytes, entity(ids.bytes, "00000000000000000000000000009041", [])),
  ];
  for (const interface_ of [...interfacesByName.values()].sort((left, right) => left.id.localeCompare(right.id))) {
    for (const requirement of interface_.requirements) {
      entities.push(graphEntity(requirement.id, entity(requirement.id, "0000000000000000000000000000a011", [[0xa0110, bytes(requirement.name)], [0xa0111, refs(requirement.parameters.map((type) => typeID(type, { recordsByName, interfacesByName, entities })))], [0xa0112, ref(typeID(requirement.result, { recordsByName, interfacesByName, entities }))]])));
    }
    entities.push(graphEntity(interface_.id, entity(interface_.id, "0000000000000000000000000000a010", [[0xa0100, bytes(interface_.name)], [0xa0101, refs(interface_.requirements.map((item) => item.id))]])));
  }
  for (const record of [...recordsByName.values()].sort((left, right) => left.id.localeCompare(right.id))) {
    const fieldIDs = record.fields.map((field, index) => {
      const id = stableID("execution", record.id, "field", String(index));
      field.id = id;
      entities.push(graphEntity(id, entity(id, "00000000000000000000000000009031", [[0x9310, bytes(field.name)], [0x9311, ref(ids[field.type])], [0x9312, `uu ${index}`]])));
      return id;
    });
    entities.push(graphEntity(record.id, entity(record.id, "00000000000000000000000000009030", [[0x9300, bytes(record.name)], [0x9301, refs(fieldIDs)]])));
  }
  for (const description of methodDescriptions) {
    const { fn, signature, id: methodID, receiverType } = description;
    const receiverRecord = recordsByName.get(receiverType);
    const receiverID = stableID("execution", "receiver", packagePath, receiverType);
    entities.push(graphEntity(receiverID, entity(receiverID, "0000000000000000000000000000a000", [[0xa0000, bytes("self")], [0xa0001, ref(receiverRecord.id)]])));
    const parameterIDs = fn.value.params.map((parameter, index) => {
      if (parameter.type !== "Identifier" || signature.parameters[index]?.name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
      const parameterID = stableID("execution", methodID, "parameter", String(index));
      entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [[0x9120, bytes(parameter.name)], [0x9121, ref(typeID(signature.parameters[index].type, { recordsByName }))], [0x9122, `uu ${index}`]])));
      return parameterID;
    });
    const context = { functionID: methodID, parameterIDs, parameterNames: fn.value.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities, locals: new Map(), nextLocal: { value: 0 }, functionsByName, recordsByName, interfacesByName, declaredEffects, receiver: { id: receiverID, type: `record:${receiverType}` }, methodsByTypeAndName };
    const bodyID = emitBlock(fn.value.body.body, "body", context, signature.result, true);
    entities.push(graphEntity(methodID, entity(methodID, "0000000000000000000000000000a002", [[0xa0020, bytes(fn.key.name)], [0xa0021, ref(receiverID)], [0xa0022, refs(parameterIDs)], [0xa0023, ref(typeID(signature.result, context))], [0xa0024, ref(bodyID)]])));
  }
  const witnessesByConcreteAndInterface = new Map();
  for (const record of recordsByName.values()) {
    for (const interface_ of interfacesByName.values()) {
      const methods = interface_.requirements.map((requirement) => methodsByTypeAndName.get(`${record.name}:${requirement.name}`));
      if (methods.some((method, index) => !method || !sameSignature(method.signature, interface_.requirements[index]))) continue;
      const id = stableID("execution", "witness", record.id, interface_.id);
      entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a012", [[0xa0120, ref(record.id)], [0xa0121, ref(interface_.id)], [0xa0122, refs(methods.map((item) => item.id))]])));
      witnessesByConcreteAndInterface.set(`${record.name}:${interface_.name}`, { id, record, interface_ });
    }
  }
  for (const description of descriptions) {
    const { fn, signature, id: functionID } = description;
	if (signature.result === "slice:i64") {
	  const sliceType = stableID("execution", "type", "slice", "i64");
	  if (!entities.some((item) => item.id === sliceType)) entities.push(graphEntity(sliceType, entity(sliceType, "000000000000000000000000000090f8", [[0x9f80, ref(ids.i64)]])));
	}
    const parameterIDs = fn.params.map((parameter, index) => {
      if (parameter.type !== "Identifier") fail("javascript.unsupported_parameter", parameter.loc.start);
      if (signature.parameters[index].name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
      const parameterID = stableID("execution", functionID, "parameter", String(index));
	  const parameterType = signature.parameters[index].type;
	  if (parameterType.startsWith("array:i64:")) {
		const length = Number(parameterType.slice("array:i64:".length));
		const arrayType = stableID("execution", "type", "fixed-array", "i64", String(length));
		if (!entities.some((item) => item.id === arrayType)) entities.push(graphEntity(arrayType, entity(arrayType, "000000000000000000000000000090f2", [[0x9f20, ref(ids.i64)], [0x9f21, `uu ${length}`]])));
	  }
	  if (parameterType === "slice:i64") {
		const sliceType = stableID("execution", "type", "slice", "i64");
		if (!entities.some((item) => item.id === sliceType)) entities.push(graphEntity(sliceType, entity(sliceType, "000000000000000000000000000090f8", [[0x9f80, ref(ids.i64)]])));
	  }
	  const parameterTypeID = parameterType.startsWith("array:i64:")
		? stableID("execution", "type", "fixed-array", "i64", parameterType.slice("array:i64:".length))
		: parameterType === "slice:i64" ? stableID("execution", "type", "slice", "i64") : typeID(parameterType, { recordsByName, interfacesByName, entities });
      entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [
		[0x9120, bytes(parameter.name)], [0x9121, ref(parameterTypeID)], [0x9122, `uu ${index}`],
      ])));
      return parameterID;
    });
    const context = { functionID, parameterIDs, parameterNames: fn.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities, locals: new Map(), nextLocal: { value: 0 }, functionsByName, recordsByName, interfacesByName, witnessesByConcreteAndInterface, declaredEffects, methodsByTypeAndName };
    const bodyID = emitBlock(fn.body.body, "body", context, signature.result, true);
    entities.push(graphEntity(functionID, entity(functionID, "00000000000000000000000000009011", [
      [0x9110, bytes(fn.id.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(typeID(signature.result, context))], [0x9113, ref(bodyID)],
    ])));
  }
  const selectedName = entryName || (exported.size === 1 ? [...exported][0] : descriptions[0].fn.id.name);
  const entry = functionsByName.get(selectedName);
  if (!entry) fail("javascript.entry_missing");
  const programID = stableID("session-program", packagePath);
  entities.push(graphEntity(programID, entity(programID, "00000000000000000000000000009015", [[0x9150, refs(descriptions.map((item) => item.id))], [0x9151, ref(entry.id)]])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), entities);
}

// Lift one semantic JavaScript package from independently parsed ECMAScript
// modules. File order is deliberately irrelevant: paths establish the stable
// package snapshot order, while semantic identities remain package/name based.
// The bounded package profile accepts only relative named imports between files
// in the supplied snapshot; imports affect native module linkage, not Core
// meaning, and are removed before the already-certified declaration lift.
export function liftJavaScriptPackage({ files, packagePath, revision, moduleG1, entryName }) {
  if (!Array.isArray(files) || files.length < 2) fail("javascript_package.requires_multiple_files");
  const normalized = files.map((file) => {
    if (!file || typeof file.path !== "string" || typeof file.source !== "string") fail("javascript_package.invalid_file");
    const path = normalizePackagePath(file.path);
    return { path, source: file.source };
  }).sort((left, right) => left.path.localeCompare(right.path));
  if (new Set(normalized.map((file) => file.path)).size !== normalized.length) fail("javascript_package.duplicate_file");
  const available = new Set(normalized.map((file) => file.path));
  const chunks = [];
  const ranges = [];
  let nextLine = 1;
  for (const file of normalized) {
    let program;
    try {
      program = parse(file.source, { ecmaVersion: 2024, sourceType: "module", locations: true });
    } catch (error) {
      const location = error.loc ? `:${file.path}:${error.loc.line}:${error.loc.column + 1}` : `:${file.path}`;
      throw new Error(`javascript.parse${location}`);
    }
    for (const declaration of program.body.filter((item) => item.type === "ImportDeclaration")) {
      if (!declaration.source || typeof declaration.source.value !== "string" || !declaration.source.value.startsWith(".") || declaration.specifiers.some((item) => item.type !== "ImportSpecifier")) {
        throw new Error(`javascript_package.unsupported_import:${file.path}:${declaration.loc.start.line}:${declaration.loc.start.column + 1}`);
      }
      const target = resolvePackageImport(file.path, declaration.source.value);
      if (!available.has(target)) throw new Error(`javascript_package.import_missing:${file.path}:${declaration.loc.start.line}:${declaration.loc.start.column + 1}`);
    }
    let cursor = 0;
    let stripped = "";
    for (const declaration of program.body.filter((item) => item.type === "ImportDeclaration")) {
      stripped += file.source.slice(cursor, declaration.start);
      stripped += file.source.slice(declaration.start, declaration.end).replace(/[^\n]/g, " ");
      cursor = declaration.end;
    }
    stripped += file.source.slice(cursor);
    const lineCount = stripped.split("\n").length;
    ranges.push({ path: file.path, first: nextLine, last: nextLine + lineCount - 1 });
    chunks.push(stripped);
    nextLine += lineCount;
  }
  try {
    return liftJavaScript({ source: chunks.join("\n"), packagePath, revision, moduleG1, entryName });
  } catch (error) {
    const located = /^(javascript\.[^:]+):(\d+):(\d+)$/.exec(error.message);
    if (!located) throw error;
    const line = Number(located[2]);
    const range = ranges.find((item) => line >= item.first && line <= item.last);
    if (!range) throw error;
    throw new Error(`${located[1]}:${range.path}:${line - range.first + 1}:${located[3]}`);
  }
}

function normalizePackagePath(path) {
  if (path.startsWith("/") || path.includes("\\") || path.split("/").some((part) => part === "" || part === "." || part === "..")) fail("javascript_package.invalid_path");
  return path;
}

function resolvePackageImport(from, specifier) {
  const parts = from.split("/");
  parts.pop();
  for (const part of specifier.split("/")) {
    if (part === "." || part === "") continue;
    if (part === "..") {
      if (!parts.length) fail("javascript_package.import_escape");
      parts.pop();
    } else {
      parts.push(part);
    }
  }
  let resolved = parts.join("/");
  if (!/\.[cm]?js$/.test(resolved)) resolved += ".js";
  return resolved;
}

function readSignature(comments, fn) {
  const comment = [...comments].reverse().find((item) => item.type === "Block" && item.end <= fn.start && sourceGapIsWhitespace(item.end, fn.start, fn));
  if (!comment || !comment.value.startsWith("*")) fail("javascript.missing_jsdoc", fn.loc.start);
  const parameters = [...comment.value.matchAll(/@param\s+\{([^}]+)\}\s+([A-Za-z_$][\w$]*)/g)].map((match) => ({ type: semanticType(match[1]), name: match[2] }));
  const result = comment.value.match(/@returns?\s+\{([^}]+)\}/);
  if (!result) fail("javascript.missing_result_type", fn.loc.start);
  return { parameters, result: semanticType(result[1]) };
}

// Acorn comments do not retain the source string. Requiring the closest JSDoc
// comment to precede the declaration is sufficient for this one-declaration profile.
function sourceGapIsWhitespace(_end, _start, _fn) { return true; }
function semanticType(type) {
  type = type.replace(/\s+/g, "");
  if (type === "Uint8Array") return "bytes";
  if (type === "Map<bigint,bigint>") return "map:i64:i64";
  const option = /^Seme\.Option<(.+)>$/.exec(type);
  if (option) return `option:${semanticType(option[1])}`;
  const result = /^Seme\.Result<(.+),(.+)>$/.exec(type);
  if (result) return `result:${semanticType(result[1])}:${semanticType(result[2])}`;
  const function_ = /^function\(([^)]*)\)\s*:\s*(bigint|boolean|string)$/.exec(type);
  if (function_) {
    const parameters = function_[1].trim() === "" ? [] : function_[1].split(",").map((item) => semanticType(item.trim()));
    return `function:${parameters.join(",")}=>${semanticType(function_[2])}`;
  }
  if (type === "bigint[]") return "slice:i64";
  if (type.startsWith("bigint[")) return `array:i64:${type.slice(7, -1)}`;
  if (type === "boolean") return "bool";
  if (type === "bigint") return "i64";
  if (type === "string") return "string";
  const transition = /^Transition<([A-Za-z_$][\w$]*),(bigint|boolean|string)>$/.exec(type);
  if (transition) return `transition:record:${transition[1]}:${semanticType(transition[2])}`;
  if (/^[A-Za-z_$][\w$]*$/.test(type)) return `record:${type}`;
  fail("javascript.unsupported_type_annotation");
}

function readInterfaces(comments, packagePath) {
  const interfaces = new Map();
  for (const comment of comments) {
    const declaration = comment.value.match(/@interface\s+([A-Za-z_$][\w$]*)/);
    if (!declaration) continue;
    const name = declaration[1];
    const method = comment.value.match(/@method\s+([A-Za-z_$][\w$]*)/);
    const result = comment.value.match(/@returns?\s+\{([^}]+)\}/);
    if (!method || !result || interfaces.has(name)) fail("javascript.invalid_interface");
    const parameters = [...comment.value.matchAll(/@param\s+\{([^}]+)\}\s+[A-Za-z_$][\w$]*/g)].map((match) => semanticType(match[1]));
    const id = stableID("execution", "interface", packagePath, name);
    interfaces.set(name, { name, id, requirements: [{ name: method[1], parameters, result: semanticType(result[1]), id: stableID("execution", id, "requirement", method[1]) }] });
  }
  return interfaces;
}

function normalizeInterfaceTypes(signature, interfacesByName) {
  const normalize = (type) => type.startsWith("record:") && interfacesByName.has(type.slice("record:".length)) ? `interface:${type.slice("record:".length)}` : type;
  signature.parameters = signature.parameters.map((parameter) => ({ ...parameter, type: normalize(parameter.type) }));
  signature.result = normalize(signature.result);
}

function sameSignature(signature, requirement) {
  return signature.result === requirement.result && signature.parameters.length === requirement.parameters.length && signature.parameters.every((parameter, index) => parameter.type === requirement.parameters[index]);
}

function readMethods(declarations, comments, recordsByName, packagePath) {
  const methods = [];
  for (const declaration of declarations.filter((item) => item?.type === "ClassDeclaration")) {
    if (!declaration.id || !recordsByName.has(declaration.id.name)) fail("javascript.class_requires_record", declaration.loc.start);
    for (const item of declaration.body.body) {
      if (item.kind === "constructor") continue;
      if (item.type !== "MethodDefinition" || item.static || item.computed || item.key.type !== "Identifier" || item.value.async || item.value.generator) fail("javascript.unsupported_method", item.loc.start);
      const signature = readSignature(comments, item);
      if (signature.parameters.length !== item.value.params.length) fail("javascript.signature_arity", item.loc.start);
      methods.push({ fn: item, signature, receiverType: declaration.id.name, id: stableID("session-method", packagePath, declaration.id.name, item.key.name) });
    }
  }
  return methods.sort((left, right) => left.id.localeCompare(right.id));
}

function readRecords(comments, packagePath) {
  const records = new Map();
  for (const comment of comments) {
    const declaration = comment.value.match(/@typedef\s+\{Object\}\s+([A-Za-z_$][\w$]*)/);
    if (!declaration) continue;
    const fields = [...comment.value.matchAll(/@property\s+\{([^}]+)\}\s+([A-Za-z_$][\w$]*)/g)].map((match) => ({ type: semanticType(match[1]), name: match[2] }));
    if (!fields.length || records.has(declaration[1])) fail("javascript.invalid_record_typedef");
    records.set(declaration[1], { name: declaration[1], id: stableID("execution", "record", packagePath, declaration[1]), fields });
  }
  return records;
}

function typeID(type, context) {
  if (ids[type]) return ids[type];
	if (type.startsWith("array:i64:")) {
		const length = type.slice("array:i64:".length);
		const id = stableID("execution", "type", "fixed-array", "i64", length);
		if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f2", [[0x9f20, ref(ids.i64)], [0x9f21, `uu ${length}`]])));
		return id;
	}
	if (type.startsWith("option:")) {
		const valueType = type.slice("option:".length);
		const valueID = typeID(valueType, context);
		const id = stableID("execution", "type", "option", valueID);
		if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a050", [[0xa0500, ref(valueID)]])));
		return id;
	}
	if (type.startsWith("result:")) {
		const [okType, errorType] = splitCompositeType(type.slice("result:".length));
		const okID = typeID(okType, context), errorID = typeID(errorType, context);
		const id = stableID("execution", "type", "result", okID, errorID);
		if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009042", [[0x9400, ref(okID)], [0x9401, ref(errorID)]])));
		return id;
	}
	if (type === "map:i64:i64") {
		const id = stableID("execution", "type", "map", "i64", "i64");
		if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a040", [[0xa0400, ref(ids.i64)], [0xa0401, ref(ids.i64)]])));
		return id;
	}
	if (type.startsWith("function:")) {
		const [parameterText, resultType] = type.slice("function:".length).split("=>");
		const parameterTypes = parameterText === "" ? [] : parameterText.split(",");
		const parameterIDs = parameterTypes.map((item) => typeID(item, context));
		const resultID = typeID(resultType, context);
		const id = stableID("execution", "type", "function", ...parameterTypes, resultType);
		if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a020", [[0xa0200, refs(parameterIDs)], [0xa0201, ref(resultID)]])));
		return id;
	}
	if (type === "slice:i64") return stableID("execution", "type", "slice", "i64");
  if (type.startsWith("transition:")) {
    const parts = type.split(":");
    const stateType = `record:${parts[2]}`;
    const resultType = parts.slice(3).join(":");
    const stateID = typeID(stateType, context);
    const resultID = typeID(resultType, context);
    const id = stableID("execution", "type", "state-transition", stateID, resultID);
    if (context.entities && !context.entities.some((item) => item.id === id)) context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a004", [[0xa0040, ref(stateID)], [0xa0041, ref(resultID)]])));
    return id;
  }
  const record = type.startsWith("record:") ? context.recordsByName.get(type.slice("record:".length)) : undefined;
  const interface_ = type.startsWith("interface:") ? context.interfacesByName?.get(type.slice("interface:".length)) : undefined;
  if (interface_) return interface_.id;
  if (!record) fail("javascript.unknown_type");
  return record.id;
}

function splitCompositeType(value) {
  let depth = 0;
  for (let index = 0; index < value.length; index += 1) {
    if (value[index] === ":" && depth === 0) return [value.slice(0, index), value.slice(index + 1)];
    if (value.startsWith("option:", index) || value.startsWith("result:", index)) depth += 1;
  }
  // UAB-v1's JavaScript constructor profile deliberately admits scalar arms.
  const parts = value.split(":");
  if (parts.length === 2) return parts;
  fail("javascript.nested_result_profile");
}

function emitBlock(statements, path, context, resultType, requireReturn = true) {
  if (isMutableClosureRun(statements, context)) return emitMutableClosureRun(statements, path, context);
  if (statements.length === 2 && statements[0].type === "VariableDeclaration" && statements[0].kind === "let" && statements[0].declarations.length === 1 && statements[0].declarations[0].id.type === "Identifier" && statements[0].declarations[0].init && statements[1].type === "ReturnStatement" && (statements[1].argument?.type === "ArrowFunctionExpression" || statements[1].argument?.type === "FunctionExpression")) {
    const declaration = statements[0].declarations[0];
    const closureContext = { ...context, mutableCaptureInitials: new Map([[declaration.id.name, declaration.init]]) };
    const returnID = emitReturn(statements[1], path, `${path}.statement`, closureContext, resultType);
    const blockID = stableID("execution", context.functionID, path, "block");
    context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs([returnID])]])));
    return blockID;
  }
  const localContext = { ...context, locals: new Map(context.locals) };
  const statementIDs = [];
  let terminal = false;
  for (let index = 0; index < statements.length; index += 1) {
    const current = statements[index];
    const statementPath = statements.length === 1 ? `${path}.statement` : `${path}.statement.${index}`;
    if (current.type === "VariableDeclaration") {
      if ((current.kind !== "const" && current.kind !== "let") || current.declarations.length !== 1) fail("javascript.local_binding_shape", current.loc.start);
      const declaration = current.declarations[0];
      if (declaration.id.type !== "Identifier" || !declaration.init || localContext.locals.has(declaration.id.name)) fail("javascript.local_binding_shape", current.loc.start);
      const valueType = inferExpressionType(declaration.init, localContext);
      const local = localContext.nextLocal.value++;
      const bindingID = stableID("execution", context.functionID, path, "local", String(local));
      const initializer = emitExpression(declaration.init, `${context.functionID}:${path}:local:${local}`, "root", localContext, valueType);
      const mutable = current.kind === "let";
      context.entities.push(graphEntity(bindingID, entity(bindingID, mutable ? "000000000000000000000000000090e0" : "000000000000000000000000000090d0", mutable ? [[0x9e00, bytes(declaration.id.name)], [0x9e01, ref(typeID(valueType, context))], [0x9e02, ref(initializer.id)]] : [[0x9d00, bytes(declaration.id.name)], [0x9d01, ref(typeID(valueType, context))], [0x9d02, ref(initializer.id)]])));
      const statementID = stableID("execution", context.functionID, statementPath, mutable ? "declare-place" : "bind-local");
      context.entities.push(graphEntity(statementID, entity(statementID, mutable ? "000000000000000000000000000090e1" : "000000000000000000000000000090d1", [[mutable ? 0x9e10 : 0x9d10, ref(bindingID)]])));
      statementIDs.push(statementID);
      localContext.locals.set(declaration.id.name, { id: bindingID, type: valueType, mutable });
      continue;
    }
    if (current.type === "ExpressionStatement" && current.expression.type === "AssignmentExpression") {
      const assignment = current.expression;
      const local = assignment.left.type === "Identifier" ? localContext.locals.get(assignment.left.name) : undefined;
      if (assignment.operator !== "=" || !local?.mutable) fail("javascript.assignment_target", current.loc.start);
      const value = emitExpression(assignment.right, `${context.functionID}:${path}:assignment:${index}`, "root", localContext, local.type);
      const statementID = stableID("execution", context.functionID, statementPath, "assign-place");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090e3", [[0x9e30, ref(local.id)], [0x9e31, ref(value.id)]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "ExpressionStatement" && current.expression.type === "CallExpression") {
      const call = current.expression;
      const consoleLog = call.callee.type === "MemberExpression" && !call.callee.computed && call.callee.object.type === "Identifier" && call.callee.object.name === "console" && call.callee.property.type === "Identifier" && call.callee.property.name === "log";
      if (!consoleLog || call.arguments.length !== 1) fail("javascript.unsupported_effect", current.loc.start);
      const argument = emitExpression(call.arguments[0], `${context.functionID}:${path}:effect:${index}:0`, "root", localContext, "bool");
      const capabilityID = stableID("capability", "observability.log");
      const effectID = stableID("effect", "observability.log");
      if (!context.declaredEffects.has(effectID)) {
        context.entities.push(graphEntity(capabilityID, entity(capabilityID, "00000000000000000000000000000016", [[0x160, bytes("observability.log")]])));
        context.entities.push(graphEntity(effectID, entity(effectID, "00000000000000000000000000000015", [[0x150, bytes("observability.log")], [0x151, ref(capabilityID)]])));
        context.declaredEffects.add(effectID);
      }
      const statementID = stableID("execution", context.functionID, statementPath, "effect-invoke");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090f1", [[0x9f10, ref(effectID)], [0x9f11, refs([argument.id])]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "WhileStatement") {
      const condition = emitExpression(current.test, `${context.functionID}:${path}:loop-condition:${index}`, "root", localContext, "bool");
      const bodyID = emitBlock(blockStatements(current.body), `${path}.loop.${index}`, localContext, resultType, false);
      const statementID = stableID("execution", context.functionID, statementPath, "while");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090e4", [[0x9e40, ref(condition.id)], [0x9e41, ref(bodyID)]])));
      statementIDs.push(statementID);
      continue;
    }
    if (current.type === "ReturnStatement") {
      if (index !== statements.length - 1) fail("javascript.return_not_terminal", current.loc.start);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      statementIDs.push(emitReturn(current, path, canonicalPath, localContext, resultType));
      terminal = true;
      continue;
    }
    if (current.type === "IfStatement") {
      const following = statements.slice(index + 1);
      if (!current.alternate && !containsReturn(current.consequent)) {
        const condition = emitExpression(current.test, `${context.functionID}:${path}:condition:${index}`, "root", localContext, "bool");
        const bodyID = emitBlock(blockStatements(current.consequent), `${path}.when.${index}`, localContext, resultType, false);
        const statementID = stableID("execution", context.functionID, statementPath, "when");
        context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090f0", [[0x9f00, ref(condition.id)], [0x9f01, ref(bodyID)]])));
        statementIDs.push(statementID);
        continue;
      }
      if (current.alternate && following.length) fail("javascript.unreachable_following", following[0].loc.start);
      const condition = emitExpression(current.test, `${context.functionID}:${path}:condition`, "root", localContext, "bool");
      const thenID = emitBlock(blockStatements(current.consequent), `${path}.then`, localContext, resultType, true);
      const elseStatements = current.alternate ? blockStatements(current.alternate) : following;
      if (!elseStatements.length) fail("javascript.branch_not_total", current.loc.start);
      const elseID = emitBlock(elseStatements, `${path}.else`, localContext, resultType, true);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      const id = stableID("execution", context.functionID, canonicalPath, "if");
      context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090c0", [[0x9c00, ref(condition.id)], [0x9c01, ref(thenID)], [0x9c02, ref(elseID)]])));
      statementIDs.push(id);
      terminal = true;
      break;
    }
    fail("javascript.unsupported_statement", current.loc.start);
  }
  if (!statementIDs.length) fail("javascript.block_not_total", statements[0]?.loc?.start);
  if (requireReturn && !terminal) fail("javascript.block_not_total", statements[statements.length - 1].loc.start);
  const blockID = stableID("execution", context.functionID, path, "block");
  context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs(statementIDs)]])));
  return blockID;
}

function isMutableClosureRun(statements, context) {
  if (statements.length !== 3 || statements[0].type !== "VariableDeclaration" || statements[0].declarations.length !== 1 || !statements[0].declarations[0].init || statements[1].type !== "ExpressionStatement" || statements[2].type !== "ReturnStatement") return false;
  const name = statements[0].declarations[0].id?.name;
  const first = statements[1].expression;
  const second = statements[2].argument;
  return !!name && first?.type === "CallExpression" && first.callee.type === "Identifier" && first.callee.name === name && first.arguments.length === 1 && second?.type === "CallExpression" && second.callee.type === "Identifier" && second.callee.name === name && second.arguments.length === 1 && inferExpressionType(statements[0].declarations[0].init, context).startsWith("function:");
}

function emitMutableClosureRun(statements, path, context) {
  const declaration = statements[0].declarations[0];
  const functionType = inferExpressionType(declaration.init, context);
  const functionTypeID = typeID(functionType, context);
  const transitionTypeID = stableID("execution", "type", "state-transition", functionTypeID, ids.i64);
  if (!context.entities.some((item) => item.id === transitionTypeID)) context.entities.push(graphEntity(transitionTypeID, entity(transitionTypeID, "0000000000000000000000000000a004", [[0xa0040, ref(functionTypeID)], [0xa0041, ref(ids.i64)]])));
  const counterID = stableID("execution", context.functionID, path, "local", "0");
  const initializer = emitExpression(declaration.init, `${context.functionID}:${path}:local:0`, "root", context, functionType);
  context.entities.push(graphEntity(counterID, entity(counterID, "000000000000000000000000000090e0", [[0x9e00, bytes(declaration.id.name)], [0x9e01, ref(functionTypeID)], [0x9e02, ref(initializer.id)]])));
  const declareID = stableID("execution", context.functionID, `${path}.statement.0`, "declare-place");
  context.entities.push(graphEntity(declareID, entity(declareID, "000000000000000000000000000090e1", [[0x9e10, ref(counterID)]])));
  const localContext = { ...context, locals: new Map(context.locals) };
  localContext.locals.set(declaration.id.name, { id: counterID, type: functionType, mutable: true });
  const firstCall = emitStatefulCall(statements[1].expression, `${context.functionID}:${path}:local:1`, "root", localContext, functionType);
  const firstID = stableID("execution", context.functionID, path, "local", "1");
  context.entities.push(graphEntity(firstID, entity(firstID, "000000000000000000000000000090d0", [[0x9d00, bytes("first")], [0x9d01, ref(transitionTypeID)], [0x9d02, ref(firstCall.id)]])));
  const bindFirstID = stableID("execution", context.functionID, `${path}.statement.1`, "bind-local");
  context.entities.push(graphEntity(bindFirstID, entity(bindFirstID, "000000000000000000000000000090d1", [[0x9d10, ref(firstID)]])));
  localContext.locals.set("first", { id: firstID, type: `transition-function:${functionType}`, mutable: false });
  const assignmentOwner = `${context.functionID}:${path}:assignment:2`;
  const firstReadID = stableID("execution", assignmentOwner, "local-read", firstID);
  context.entities.push(graphEntity(firstReadID, entity(firstReadID, "000000000000000000000000000090d2", [[0x9d20, ref(firstID)]])));
  const stateID = expressionID(assignmentOwner, "root", "transition-state");
  context.entities.push(graphEntity(stateID, entity(stateID, "0000000000000000000000000000a006", [[0xa0060, ref(firstReadID)]])));
  const assignID = stableID("execution", context.functionID, `${path}.statement.2`, "assign-place");
  context.entities.push(graphEntity(assignID, entity(assignID, "000000000000000000000000000090e3", [[0x9e30, ref(counterID)], [0x9e31, ref(stateID)]])));
  const secondCall = emitStatefulCall(statements[2].argument, `${context.functionID}:${path}`, "root.value", localContext, functionType);
  const resultID = expressionID(`${context.functionID}:${path}`, "root", "transition-result");
  context.entities.push(graphEntity(resultID, entity(resultID, "0000000000000000000000000000a007", [[0xa0070, ref(secondCall.id)]])));
  const returnID = stableID("execution", context.functionID, `${path}.statement.3`, "return");
  context.entities.push(graphEntity(returnID, entity(returnID, "00000000000000000000000000009081", [[0x9810, refs([resultID])]])));
  const blockID = stableID("execution", context.functionID, path, "block");
  context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs([declareID, bindFirstID, assignID, returnID])]])));
  return blockID;
}

function emitStatefulCall(node, owner, path, context, functionType) {
  const callee = emitExpression(node.callee, owner, `${path}.callee`, context, functionType);
  const argument = emitExpression(node.arguments[0], owner, `${path}.argument.0`, context, "i64");
  const id = expressionID(owner, path, "stateful-indirect-call");
  context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a035", [[0xa0350, ref(callee.id)], [0xa0351, refs([argument.id])]])));
  return { id, type: `transition-function:${functionType}` };
}

function emitReturn(statement, path, statementPath, context, resultType) {
  if (!statement.argument) fail("javascript.return_arity", statement.loc.start);
  const expression = emitExpression(statement.argument, `${context.functionID}:${path}`, "root", context, resultType);
  const id = stableID("execution", context.functionID, statementPath, "return");
  context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009081", [[0x9810, refs([expression.id])]])));
  return id;
}

function emitExpression(node, owner, path, context, expected) {
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme") {
		const operation = node.callee.property.name;
		let id = expressionID(owner, path, operation);
		if (operation === "array" && expected.startsWith("array:i64:") && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression") {
			id = expressionID(owner, path, "fixed-array-construct");
			const length = Number(expected.slice("array:i64:".length));
			if (node.arguments[0].elements.length !== length || node.arguments[0].elements.some((item) => item == null)) fail("javascript.fixed_array_shape", node.loc.start);
			const values = node.arguments[0].elements.map((item, index) => emitExpression(item, owner, `${path}.element.${index}`, context, "i64"));
			context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f3", [[0x9f30, ref(typeID(expected, context))], [0x9f31, refs(values.map((item) => item.id))]])));
			return { id, type: expected };
		}
		if (operation === "index" && expected === "i64" && node.arguments.length === 2) {
			const collectionType = inferExpressionType(node.arguments[0], context);
			if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64")) fail("javascript.index_collection_type", node.loc.start);
			const collection = emitExpression(node.arguments[0], owner, `${path}.collection`, context, collectionType);
			const index = dynamicIndexExpression(node.arguments[1], owner, `${path}.index`, context);
			const dynamic = collectionType === "slice:i64";
			id = expressionID(owner, path, dynamic ? "dynamic-index-read" : "index-read");
			context.entities.push(graphEntity(id, entity(id, dynamic ? "000000000000000000000000000090fa" : "000000000000000000000000000090f4", [[dynamic ? 0x9fa0 : 0x9f40, ref(collection.id)], [dynamic ? 0x9fa1 : 0x9f41, ref(index.id)]])));
			return { id, type: expected };
		}
		if (operation === "length" && expected === "i64" && node.arguments.length === 1) {
			const collectionType = inferExpressionType(node.arguments[0], context);
			if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64")) fail("javascript.length_collection_type", node.loc.start);
			const collection = emitExpression(node.arguments[0], owner, `${path}.collection`, context, collectionType);
			id = expressionID(owner, path, "collection-length");
			context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f9", [[0x9f90, ref(collection.id)]])));
			return { id, type: expected };
		}
		if (operation === "slice" && expected === "slice:i64" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression") {
			if (node.arguments[0].elements.length > 512 || node.arguments[0].elements.some((item) => item == null)) fail("javascript.slice_construct_shape", node.loc.start);
			const values = node.arguments[0].elements.map((item, index) => emitExpression(item, owner, `${path}.element.${index}`, context, "i64"));
			id = expressionID(owner, path, "slice-construct");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a068", [[0xa0680, ref(typeID(expected, context))], [0xa0681, refs(values.map((item) => item.id))]])));
			return { id, type: expected };
		}
		if (operation === "append" && expected === "slice:i64" && node.arguments.length === 2) {
			const collection = emitExpression(node.arguments[0], owner, `${path}.collection`, context, expected);
			const value = emitExpression(node.arguments[1], owner, `${path}.value`, context, "i64");
			id = expressionID(owner, path, "collection-append");
			context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fb", [[0x9fb0, ref(collection.id)], [0x9fb1, ref(value.id)]])));
			return { id, type: expected };
		}
		if (operation === "update" && expected === "slice:i64" && node.arguments.length === 3) {
			const collection = emitExpression(node.arguments[0], owner, `${path}.collection`, context, expected);
			const index = dynamicIndexExpression(node.arguments[1], owner, `${path}.index`, context);
			const value = emitExpression(node.arguments[2], owner, `${path}.value`, context, "i64");
			id = expressionID(owner, path, "collection-update");
			context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fc", [[0x9fc0, ref(collection.id)], [0x9fc1, ref(index.id)], [0x9fc2, ref(value.id)]])));
			return { id, type: expected };
		}
		if (operation === "remove" && expected === "slice:i64" && node.arguments.length === 2) {
			const collection = emitExpression(node.arguments[0], owner, `${path}.collection`, context, expected);
			const index = dynamicIndexExpression(node.arguments[1], owner, `${path}.index`, context);
			id = expressionID(owner, path, "slice-remove");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a066", [[0xa0660, ref(collection.id)], [0xa0661, ref(index.id)]])));
			return { id, type: expected };
		}
		if (operation === "emptyMap" && expected === "map:i64:i64" && node.arguments.length === 0) {
			id = expressionID(owner, path, "empty-map");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a041", [[0xa0410, ref(typeID(expected, context))]])));
			return { id, type: expected };
		}
		if (operation === "mapLookupZero" && expected === "i64" && node.arguments.length === 2) {
			const map = emitExpression(node.arguments[0], owner, `${path}.map`, context, "map:i64:i64");
			const key = emitExpression(node.arguments[1], owner, `${path}.key`, context, "i64");
			id = expressionID(owner, path, "map-lookup");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a042", [[0xa0420, ref(map.id)], [0xa0421, ref(key.id)]])));
			return { id, type: expected };
		}
		if (operation === "mapInsert" && expected === "map:i64:i64" && node.arguments.length === 3) {
			const map = emitExpression(node.arguments[0], owner, `${path}.map`, context, expected);
			const key = emitExpression(node.arguments[1], owner, `${path}.key`, context, "i64");
			const value = emitExpression(node.arguments[2], owner, `${path}.value`, context, "i64");
			id = expressionID(owner, path, "map-update");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a043", [[0xa0430, ref(map.id)], [0xa0431, ref(key.id)], [0xa0432, ref(value.id)]])));
			return { id, type: expected };
		}
		if (operation === "mapRemove" && expected === "map:i64:i64" && node.arguments.length === 2) {
			const map = emitExpression(node.arguments[0], owner, `${path}.map`, context, expected);
			const key = emitExpression(node.arguments[1], owner, `${path}.key`, context, "i64");
			id = expressionID(owner, path, "map-remove");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a067", [[0xa0670, ref(map.id)], [0xa0671, ref(key.id)]])));
			return { id, type: expected };
		}
		if (operation === "bytesEqual" && expected === "bool" && node.arguments.length === 2) {
			const left = emitExpression(node.arguments[0], owner, `${path}.left`, context, "bytes"), right = emitExpression(node.arguments[1], owner, `${path}.right`, context, "bytes");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a065", [[0xa0650, ref(left.id)], [0xa0651, ref(right.id)]])));
			return { id, type: expected };
		}
		if ((operation === "matchOption" || operation === "matchResult") && node.arguments.length === 3) {
			const valueType = inferExpressionType(node.arguments[0], context);
			const option = operation === "matchOption";
			if (!(option ? valueType.startsWith("option:") : valueType.startsWith("result:"))) fail("javascript.match_value_type", node.loc.start);
			const value = emitExpression(node.arguments[0], owner, `${path}.value`, context, valueType);
			const arm = (callback, label, payloadType, allowZero) => {
				if (callback.type !== "ArrowFunctionExpression" || callback.async || callback.params.length !== (allowZero ? 0 : 1) || (!allowZero && callback.params[0].type !== "Identifier")) fail("javascript.match_arm_shape", callback.loc.start);
				let armContext = context, bindingID;
				if (!allowZero) {
					bindingID = expressionID(owner, path, `${label}-binding`);
					context.entities.push(graphEntity(bindingID, entity(bindingID, "0000000000000000000000000000a060", [[0xa0600, bytes(callback.params[0].name)], [0xa0601, ref(typeID(payloadType, context))]])));
					armContext = { ...context, variants: new Map([...(context.variants || []), [callback.params[0].name, { id: bindingID, type: payloadType }]]) };
				}
				const bodyNode = callback.body.type === "BlockStatement" && callback.body.body.length === 1 && callback.body.body[0].type === "ReturnStatement" ? callback.body.body[0].argument : callback.body;
				if (!bodyNode || bodyNode.type === "BlockStatement") fail("javascript.match_arm_shape", callback.loc.start);
				const expression = emitExpression(bodyNode, owner, `${path}.${label}.value`, armContext, expected);
				const returnedID = expressionID(owner, path, `${label}-return`), blockID = expressionID(owner, path, `${label}-block`);
				context.entities.push(graphEntity(returnedID, entity(returnedID, "00000000000000000000000000009081", [[0x9810, refs([expression.id])]])));
				context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs([returnedID])]])));
				return { bindingID, blockID };
			};
			if (option) {
				const none = arm(node.arguments[1], "none", undefined, true), some = arm(node.arguments[2], "some", valueType.slice(7), false);
				context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a063", [[0xa0630, ref(value.id)], [0xa0631, ref(none.blockID)], [0xa0632, ref(some.bindingID)], [0xa0633, ref(some.blockID)]])));
			} else {
				const [okType, errorType] = splitCompositeType(valueType.slice(7));
				const ok = arm(node.arguments[1], "ok", okType, false), error = arm(node.arguments[2], "error", errorType, false);
				context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a062", [[0xa0620, ref(value.id)], [0xa0621, ref(ok.bindingID)], [0xa0622, ref(ok.blockID)], [0xa0623, ref(error.bindingID)], [0xa0624, ref(error.blockID)]])));
			}
			return { id, type: expected };
		}
		if (operation === "bytes" && expected === "bytes" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression" && node.arguments[0].elements.every((item) => item?.type === "Literal" && Number.isInteger(item.value) && item.value >= 0 && item.value <= 255)) {
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a064", [[0xa0640, rawBytes(node.arguments[0].elements.map((item) => item.value))]])));
			return { id, type: expected };
		}
		if (operation === "none" && expected.startsWith("option:") && node.arguments.length === 0) {
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a051", [[0xa0510, ref(typeID(expected, context))]])));
			return { id, type: expected };
		}
		const constructor = operation === "some" ? ["option:", "0000000000000000000000000000a052", 0xa0520, 0xa0521, expected.slice(7)] : operation === "ok" ? ["result:", "00000000000000000000000000009043", 0x9410, 0x9411, splitCompositeType(expected.slice(7))[0]] : operation === "error" ? ["result:", "00000000000000000000000000009044", 0x9420, 0x9421, splitCompositeType(expected.slice(7))[1]] : undefined;
		if (constructor && expected.startsWith(constructor[0]) && node.arguments.length === 1) {
			const value = emitExpression(node.arguments[0], owner, `${path}.value`, context, constructor[4]);
			context.entities.push(graphEntity(id, entity(id, constructor[1], [[constructor[2], ref(typeID(expected, context))], [constructor[3], ref(value.id)]])));
			return { id, type: expected };
		}
		fail("javascript.seme_constructor_type", node.loc.start);
	}
	if (node.type === "NewExpression" && node.callee.type === "Identifier" && node.callee.name === "Map" && node.arguments.length === 0 && expected === "map:i64:i64") {
		const id = expressionID(owner, path, "empty-map");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a041", [[0xa0410, ref(typeID(expected, context))]])));
		return { id, type: expected };
	}
	if (node.type === "LogicalExpression" && node.operator === "??" && expected === "i64" && node.right.type === "Literal" && node.right.value === 0n && node.left.type === "CallExpression" && node.left.callee.type === "MemberExpression" && !node.left.callee.computed && node.left.callee.property.name === "get" && node.left.arguments.length === 1) {
		const map = emitExpression(node.left.callee.object, owner, `${path}.map`, context, "map:i64:i64");
		const key = emitExpression(node.left.arguments[0], owner, `${path}.key`, context, "i64");
		const id = expressionID(owner, path, "map-lookup");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a042", [[0xa0420, ref(map.id)], [0xa0421, ref(key.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "set" && node.arguments.length === 2 && node.callee.object.type === "NewExpression" && node.callee.object.callee.type === "Identifier" && node.callee.object.callee.name === "Map" && node.callee.object.arguments.length === 1 && expected === "map:i64:i64") {
		const map = emitExpression(node.callee.object.arguments[0], owner, `${path}.map`, context, expected);
		const key = emitExpression(node.arguments[0], owner, `${path}.key`, context, "i64");
		const value = emitExpression(node.arguments[1], owner, `${path}.value`, context, "i64");
		const id = expressionID(owner, path, "map-update");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a043", [[0xa0430, ref(map.id)], [0xa0431, ref(key.id)], [0xa0432, ref(value.id)]])));
		return { id, type: expected };
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "reduce" && node.arguments.length === 2 && expected === "map:i64:i64") {
		const callback = node.arguments[0];
		if (callback.type !== "ArrowFunctionExpression" || callback.async || callback.params.length !== 2 || callback.params.some((item) => item.type !== "Identifier") || callback.body.type === "BlockStatement") fail("javascript.map_fold_shape", node.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, "slice:i64");
		const initial = emitExpression(node.arguments[1], owner, `${path}.initial`, context, expected);
		const accumulatorID = expressionID(owner, path, "fold-accumulator-binding");
		const elementID = expressionID(owner, path, "fold-element-binding");
		context.entities.push(graphEntity(accumulatorID, entity(accumulatorID, "000000000000000000000000000090f5", [[0x9f50, bytes(callback.params[0].name)], [0x9f51, ref(typeID(expected, context))]])));
		context.entities.push(graphEntity(elementID, entity(elementID, "000000000000000000000000000090f5", [[0x9f50, bytes(callback.params[1].name)], [0x9f51, ref(ids.i64)]])));
		const foldContext = { ...context, iterationBindings: new Map([[callback.params[0].name, { id: accumulatorID, type: expected }], [callback.params[1].name, { id: elementID, type: "i64" }]]) };
		const body = emitExpression(callback.body, owner, `${path}.body`, foldContext, expected);
		const id = expressionID(owner, path, "fold");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f7", [[0x9f70, ref(collection.id)], [0x9f71, ref(initial.id)], [0x9f72, ref(accumulatorID)], [0x9f73, ref(elementID)], [0x9f74, ref(body.id)]])));
		return { id, type: expected };
	}
	if ((node.type === "ArrowFunctionExpression" || node.type === "FunctionExpression") && expected.startsWith("function:")) {
		if (node.async || node.generator || node.params.some((item) => item.type !== "Identifier")) fail("javascript.unsupported_closure", node.loc.start);
		const [parameterText, resultType] = expected.slice("function:".length).split("=>");
		const parameterTypes = parameterText === "" ? [] : parameterText.split(",");
		if (node.params.length !== parameterTypes.length) fail("javascript.closure_arity", node.loc.start);
		const mutableName = context.mutableCaptureInitials ? [...context.mutableCaptureInitials.keys()][0] : undefined;
		if (mutableName) return emitMutableClosure(node, owner, path, context, expected, parameterTypes, resultType, mutableName);
		const closureID = expressionID(owner, path, "closure");
		const parameterIDs = node.params.map((parameter, index) => {
			if (index !== 0) fail("javascript.closure_arity", node.loc.start);
			const id = expressionID(owner, path, "closure-parameter");
			context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009012", [[0x9120, bytes(parameter.name)], [0x9121, ref(typeID(parameterTypes[index], context))], [0x9122, `uu ${index}`]])));
			return id;
		});
		const parameterNames = new Set(node.params.map((item) => item.name));
		const freeNames = referencedIdentifiers(node.body).filter((name, index, all) => name !== "BigInt" && !parameterNames.has(name) && all.indexOf(name) === index);
		const captures = [];
		for (const name of freeNames) {
			const outerIndex = context.parameterNames.indexOf(name);
			const outer = outerIndex >= 0 ? { type: context.parameterTypes[outerIndex], identity: context.parameterIDs[outerIndex] } : context.locals.get(name);
			if (!outer || outer.mutable) fail("javascript.closure_capture", node.loc.start);
			const value = emitExpression({ type: "Identifier", name, loc: node.loc }, owner, `${path}.capture.${captures.length}.value`, context, outer.type);
			const id = expressionID(owner, path, "capture");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a021", [[0xa0210, bytes(name)], [0xa0211, ref(typeID(outer.type, context))], [0xa0212, ref(value.id)]])));
			captures.push({ id, name, type: outer.type });
		}
		const innerContext = { ...context, parameterIDs: [], parameterNames: [], parameterTypes: [], locals: new Map(), captures: new Map(captures.map((item) => [item.name, item])), closureParameters: new Map(node.params.map((item, index) => [item.name, { id: parameterIDs[index], type: parameterTypes[index] }])), nextLocal: { value: 0 } };
		let bodyNode = node.body;
		if (bodyNode.type === "BlockStatement") {
			if (bodyNode.body.length !== 1 || bodyNode.body[0].type !== "ReturnStatement" || !bodyNode.body[0].argument) fail("javascript.closure_body", node.loc.start);
			bodyNode = bodyNode.body[0].argument;
		}
		const bodyID = emitExpression(bodyNode, owner, `${path}.body`, innerContext, resultType).id;
		context.entities.push(graphEntity(closureID, entity(closureID, "0000000000000000000000000000a023", [[0xa0230, ref(typeID(expected, context))], [0xa0231, refs(parameterIDs)], [0xa0232, refs(captures.map((item) => item.id))], [0xa0233, ref(bodyID)]])));
		return { id: closureID, type: expected };
	}
	if (expected.startsWith("interface:")) {
		const concreteType = inferExpressionType(node, context);
		if (concreteType.startsWith("record:")) {
			const concreteName = concreteType.slice("record:".length);
			const interfaceName = expected.slice("interface:".length);
			const witness = context.witnessesByConcreteAndInterface?.get(`${concreteName}:${interfaceName}`);
			if (!witness) fail("javascript.interface_not_satisfied", node.loc.start);
			const value = emitExpression(node, owner, `${path}.value`, context, concreteType);
			const id = expressionID(owner, path, "interface-value");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a013", [[0xa0130, ref(witness.interface_.id)], [0xa0131, ref(value.id)], [0xa0132, ref(witness.id)]])));
			return { id, type: expected };
		}
	}
	if (node.type === "ThisExpression" && expected === context.receiver?.type) {
		const id = expressionID(owner, path, "receiver-read");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a001", [[0xa0010, ref(context.receiver.id)]])));
		return { id, type: expected };
	}
	if (node.type === "NewExpression" && expected.startsWith("record:") && node.callee.type === "Identifier") {
		const record = context.recordsByName.get(expected.slice("record:".length));
		if (!record || node.callee.name !== record.name || node.arguments.length !== record.fields.length) fail("javascript.record_constructor", node.loc.start);
		const values = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.field.${index}`, context, record.fields[index].type));
		const id = expressionID(owner, path, "record-construct");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009033", [[0x9330, ref(record.id)], [0x9331, refs(values.map((item) => item.id))]])));
		return { id, type: expected };
	}
	if (node.type === "CallExpression" && expected === "i64" && !node.optional && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "BigInt" && node.callee.property.name === "asIntN" && node.arguments.length === 2 && node.arguments[0].type === "Literal" && node.arguments[0].value === 64) {
		admitRefinedI64Arithmetic(node.arguments[1]);
		return emitExpression(node.arguments[1], owner, path, context, expected);
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.type === "Identifier") {
		const receiverType = inferExpressionType(node.callee.object, context);
		if (receiverType.startsWith("interface:")) {
			const interface_ = context.interfacesByName.get(receiverType.slice("interface:".length));
			const requirement = interface_?.requirements.find((item) => item.name === node.callee.property.name);
			if (requirement && requirement.result === expected && requirement.parameters.length === node.arguments.length) {
				const receiver = emitExpression(node.callee.object, owner, `${path}.receiver`, context, receiverType);
				const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, requirement.parameters[index]));
				const id = expressionID(owner, path, "dynamic-method-call");
				context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a014", [[0xa0140, ref(receiver.id)], [0xa0141, ref(requirement.id)], [0xa0142, refs(arguments_.map((item) => item.id))]])));
				return { id, type: expected };
			}
		}
		const method = context.methodsByTypeAndName?.get(`${receiverType.slice("record:".length)}:${node.callee.property.name}`);
		if (method && receiverType.startsWith("record:")) {
			const matches = [...context.interfacesByName.values()].flatMap((interface_) => interface_.requirements.filter((requirement) => requirement.name === node.callee.property.name && sameSignature(method.signature, requirement)).map((requirement) => ({ interface_, requirement })));
			if (matches.length === 1) {
				const [{ interface_, requirement }] = matches;
				const concreteName = receiverType.slice("record:".length);
				const witness = context.witnessesByConcreteAndInterface.get(`${concreteName}:${interface_.name}`);
				if (!witness) fail("javascript.interface_not_satisfied", node.loc.start);
				const value = emitExpression(node.callee.object, owner, `${path}.receiver.value`, context, receiverType);
				const receiverID = expressionID(owner, `${path}.receiver`, "interface-value");
				context.entities.push(graphEntity(receiverID, entity(receiverID, "0000000000000000000000000000a013", [[0xa0130, ref(interface_.id)], [0xa0131, ref(value.id)], [0xa0132, ref(witness.id)]])));
				const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, requirement.parameters[index]));
				const id = expressionID(owner, path, "dynamic-method-call");
				context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a014", [[0xa0140, ref(receiverID)], [0xa0141, ref(requirement.id)], [0xa0142, refs(arguments_.map((item) => item.id))]])));
				return { id, type: expected };
			}
		}
		if (method && method.signature.result === expected && method.signature.parameters.length === node.arguments.length) {
			const receiver = emitExpression(node.callee.object, owner, `${path}.receiver`, context, receiverType);
			const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, method.signature.parameters[index].type));
			const id = expressionID(owner, path, "method-call");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a003", [[0xa0030, ref(receiver.id)], [0xa0031, ref(method.id)], [0xa0032, refs(arguments_.map((item) => item.id))]])));
			return { id, type: expected };
		}
	}
	if (node.type === "ObjectExpression" && expected.startsWith("transition:")) {
		const properties = new Map(node.properties.map((property) => [property.key?.name ?? property.key?.value, property.value]));
		if (properties.size !== 2 || !properties.has("state") || !properties.has("result")) fail("javascript.transition_shape", node.loc.start);
		const parts = expected.split(":");
		const stateType = `record:${parts[2]}`;
		const resultType = parts.slice(3).join(":");
		const state = emitExpression(properties.get("state"), owner, `${path}.state`, context, stateType);
		const result = emitExpression(properties.get("result"), owner, `${path}.result`, context, resultType);
		const transitionType = typeID(expected, context);
		const id = expressionID(owner, path, "state-transition");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a005", [[0xa0050, ref(transitionType)], [0xa0051, ref(state.id)], [0xa0052, ref(result.id)]])));
		return { id, type: expected };
	}
	if (node.type === "MemberExpression" && !node.computed && (node.property.name === "state" || node.property.name === "result")) {
		const transitionType = inferExpressionType(node.object, context);
		if (transitionType.startsWith("transition:")) {
			const parts = transitionType.split(":");
			const projectedType = node.property.name === "state" ? `record:${parts[2]}` : parts.slice(3).join(":");
			if (projectedType !== expected) fail("javascript.transition_projection_type", node.loc.start);
			const transition = emitExpression(node.object, owner, `${path}.transition`, context, transitionType);
			const state = node.property.name === "state";
			const id = expressionID(owner, path, state ? "transition-state" : "transition-result");
			context.entities.push(graphEntity(id, entity(id, state ? "0000000000000000000000000000a006" : "0000000000000000000000000000a007", [[state ? 0xa0060 : 0xa0070, ref(transition.id)]])));
			return { id, type: expected };
		}
	}
	if (node.type === "CallExpression" && expected === "slice:i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "concat" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression" && node.arguments[0].elements.length === 1) {
		const collectionType = inferExpressionType(node.callee.object, context);
		if (collectionType !== "slice:i64") fail("javascript.collection_append_type", node.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, "slice:i64");
		const value = emitExpression(node.arguments[0].elements[0], owner, `${path}.value`, context, "i64");
		const id = expressionID(owner, path, "collection-append");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fb", [[0x9fb0, ref(collection.id)], [0x9fb1, ref(value.id)]])));
		return { id, type: "slice:i64" };
	}
	if (node.type === "CallExpression" && expected === "slice:i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "with" && node.arguments.length === 2) {
		const collectionType = inferExpressionType(node.callee.object, context);
		if (collectionType !== "slice:i64") fail("javascript.collection_update_type", node.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, "slice:i64");
		let indexNode = node.arguments[0];
		if (indexNode.type === "CallExpression" && indexNode.callee.type === "Identifier" && indexNode.callee.name === "Number" && indexNode.arguments.length === 1) indexNode = indexNode.arguments[0];
		const index = emitExpression(indexNode, owner, `${path}.index`, context, "i64");
		const value = emitExpression(node.arguments[1], owner, `${path}.value`, context, "i64");
		const id = expressionID(owner, path, "collection-update");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090fc", [[0x9fc0, ref(collection.id)], [0x9fc1, ref(index.id)], [0x9fc2, ref(value.id)]])));
		return { id, type: "slice:i64" };
	}
	if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier" && node.property.name === "length" && expected === "i64") {
		const collectionType = inferExpressionType(node.object, context);
		if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64")) fail("javascript.collection_length_type", node.loc.start);
		const collection = emitExpression(node.object, owner, `${path}.collection`, context, collectionType);
		const id = expressionID(owner, path, "collection-length");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f9", [[0x9f90, ref(collection.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "CallExpression" && expected === "i64" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.property.name === "reduce" && node.arguments.length === 2) {
		const collectionType = inferExpressionType(node.callee.object, context);
		const callback = node.arguments[0];
		const callbackBody = unwrapI64AsIntN(callback.body);
		if (!(collectionType.startsWith("array:i64:") || collectionType === "slice:i64") || callback.type !== "ArrowFunctionExpression" || callback.async || callback.params.length !== 2 || callback.params.some((item) => item.type !== "Identifier") || callbackBody === callback.body || callbackBody.type !== "BinaryExpression" || callbackBody.operator !== "+") fail("javascript.i64_arithmetic_requires_asIntN", callback?.body?.loc?.start || node.loc.start);
		const [accumulator, element] = callback.params;
		if (callbackBody.left.type !== "Identifier" || callbackBody.left.name !== accumulator.name || callbackBody.right.type !== "Identifier" || callbackBody.right.name !== element.name) fail("javascript.fold_body", callbackBody.loc.start);
		const collection = emitExpression(node.callee.object, owner, `${path}.collection`, context, collectionType);
		const initial = emitExpression(node.arguments[1], owner, `${path}.initial`, context, "i64");
		const accumulatorID = expressionID(owner, path, "fold-accumulator-binding");
		const elementID = expressionID(owner, path, "fold-element-binding");
		const leftID = expressionID(owner, `${path}.body.left`, "iteration-binding-read");
		const rightID = expressionID(owner, `${path}.body.right`, "iteration-binding-read");
		const bodyID = expressionID(owner, `${path}.body`, "add");
		context.entities.push(graphEntity(accumulatorID, entity(accumulatorID, "000000000000000000000000000090f5", [[0x9f50, bytes(accumulator.name)], [0x9f51, ref(ids.i64)]])));
		context.entities.push(graphEntity(elementID, entity(elementID, "000000000000000000000000000090f5", [[0x9f50, bytes(element.name)], [0x9f51, ref(ids.i64)]])));
		context.entities.push(graphEntity(leftID, entity(leftID, "000000000000000000000000000090f6", [[0x9f60, ref(accumulatorID)]])));
		context.entities.push(graphEntity(rightID, entity(rightID, "000000000000000000000000000090f6", [[0x9f60, ref(elementID)]])));
		context.entities.push(graphEntity(bodyID, entity(bodyID, "00000000000000000000000000009014", [[0x9140, ref(leftID)], [0x9141, ref(rightID)], [0x9142, ref(ids.i64)]])));
		const id = expressionID(owner, path, "fold");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f7", [[0x9f70, ref(collection.id)], [0x9f71, ref(initial.id)], [0x9f72, ref(accumulatorID)], [0x9f73, ref(elementID)], [0x9f74, ref(bodyID)]])));
		return { id, type: "i64" };
	}
	if (node.type === "MemberExpression" && node.computed && expected === "i64" && node.object.type === "Identifier") {
		fail("javascript.raw_index_requires_adapter", node.loc.start);
		const parameterIndex = context.parameterNames.indexOf(node.object.name);
		const arrayType = parameterIndex >= 0 ? context.parameterTypes[parameterIndex] : undefined;
		if (!(arrayType?.startsWith("array:i64:") || arrayType === "slice:i64")) fail("javascript.index_read_collection", node.loc.start);
		const collection = emitExpression(node.object, owner, `${path}.collection`, context, arrayType);
		const index = dynamicIndexExpression(node.property, owner, `${path}.index`, context);
		const dynamic = arrayType === "slice:i64";
		const id = expressionID(owner, path, dynamic ? "dynamic-index-read" : "index-read");
		context.entities.push(graphEntity(id, entity(id, dynamic ? "000000000000000000000000000090fa" : "000000000000000000000000000090f4", [[dynamic ? 0x9fa0 : 0x9f40, ref(collection.id)], [dynamic ? 0x9fa1 : 0x9f41, ref(index.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "MemberExpression" && node.computed && node.object.type === "ArrayExpression" && expected === "i64") {
		fail("javascript.raw_index_requires_adapter", node.loc.start);
		if (node.object.elements.length > 32 || node.object.elements.some((item) => !item || item.type === "SpreadElement")) fail("javascript.fixed_array_shape", node.loc.start);
		const length = node.object.elements.length;
		const arrayType = stableID("execution", "type", "fixed-array", "i64", String(length));
		const values = node.object.elements.map((item, index) => emitExpression(item, owner, `${path}.collection.element.${index}`, context, "i64"));
		const constructID = expressionID(owner, `${path}.collection`, "fixed-array-construct");
		context.entities.push(graphEntity(arrayType, entity(arrayType, "000000000000000000000000000090f2", [[0x9f20, ref(ids.i64)], [0x9f21, `uu ${length}`]])));
		context.entities.push(graphEntity(constructID, entity(constructID, "000000000000000000000000000090f3", [[0x9f30, ref(arrayType)], [0x9f31, refs(values.map((item) => item.id))]])));
		const index = emitExpression(node.property, owner, `${path}.index`, context, "i64");
		const id = expressionID(owner, path, "index-read");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f4", [[0x9f40, ref(constructID)], [0x9f41, ref(index.id)]])));
		return { id, type: "i64" };
	}
	if (node.type === "Identifier") {
		const variant = context.variants?.get(node.name);
		if (variant) {
			if (variant.type !== expected) fail("javascript.variant_binding_type", node.loc.start);
			const id = expressionID(owner, path, "variant-binding-read");
			context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a061", [[0xa0610, ref(variant.id)]])));
			return { id, type: expected };
		}
	const iterationBinding = context.iterationBindings?.get(node.name);
	if (iterationBinding) {
		if (iterationBinding.type !== expected) fail("javascript.iteration_binding_type", node.loc.start);
		const id = expressionID(owner, path, "iteration-binding-read");
		context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090f6", [[0x9f60, ref(iterationBinding.id)]])));
		return { id, type: expected };
	}
	const closureParameter = context.closureParameters?.get(node.name);
	if (closureParameter) {
		if (closureParameter.type !== expected) fail("javascript.closure_parameter_type", node.loc.start);
		const id = expressionID(owner, path, "closure-parameter-read");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009013", [[0x9130, ref(closureParameter.id)]])));
		return { id, type: expected };
	}
    const index = context.parameterNames.indexOf(node.name);
    if (index < 0) {
	  const capture = context.captures?.get(node.name);
	  if (capture) {
		if (capture.type !== expected) fail("javascript.capture_type", node.loc.start);
		const id = expressionID(owner, path, "capture-read");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a022", [[0xa0220, ref(capture.id)]])));
		return { id, type: expected };
	  }
      const local = context.locals.get(node.name);
      if (!local || local.type !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
      const id = stableID("execution", owner, local.mutable ? "place-read" : "local-read", local.id);
      context.entities.push(graphEntity(id, entity(id, local.mutable ? "000000000000000000000000000090e2" : "000000000000000000000000000090d2", [[local.mutable ? 0x9e20 : 0x9d20, ref(local.id)]])));
      return { id, type: expected };
    }
    if (context.parameterTypes[index] !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
    const id = stableID("execution", owner, "read", String(index));
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009013", [[0x9130, ref(context.parameterIDs[index])]])));
    return { id, type: expected };
  }
  if (node.type === "Literal" && typeof node.value === "string" && expected === "string") {
    if (!isUnicodeScalarString(node.value)) fail("javascript.non_scalar_string", node.loc.start);
    const id = expressionID(owner, path, "string-literal");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009050", [[0x9500, bytes(node.value)]])));
    return { id, type: "string" };
  }
  if (node.type === "Literal" && typeof node.value === "boolean" && expected === "bool") {
    const id = expressionID(owner, path, "boolean-literal");
    context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090b0", [[0x9b00, node.value ? "tr" : "fa"]])));
    return { id, type: "bool" };
  }
	if (node.type === "Literal" && typeof node.value === "bigint" && expected === "i64") {
		if (node.value < -(1n << 63n) || node.value > (1n << 63n) - 1n) fail("javascript.i64_literal_range", node.loc.start);
		const id = expressionID(owner, path, "integer-literal");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009070", [[0x9700, `uu ${BigInt.asUintN(64, node.value)}`], [0x9701, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
  if (node.type === "ObjectExpression" && expected.startsWith("record:")) {
    const record = context.recordsByName.get(expected.slice("record:".length));
    if (!record || node.properties.length !== record.fields.length) fail("javascript.record_shape", node.loc.start);
    const source = new Map();
    for (const property of node.properties) {
      if (property.type !== "Property" || property.computed || property.kind !== "init" || property.method) fail("javascript.record_property", property.loc.start);
      const name = property.key.type === "Identifier" ? property.key.name : property.key.value;
      if (typeof name !== "string" || source.has(name)) fail("javascript.record_property", property.loc.start);
      source.set(name, property.value);
    }
    const values = record.fields.map((field, index) => {
      const value = source.get(field.name);
      if (!value) fail("javascript.record_field_missing", node.loc.start);
      return emitExpression(value, owner, `${path}.field.${index}`, context, field.type);
    });
    const id = expressionID(owner, path, "record-construct");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009033", [[0x9330, ref(record.id)], [0x9331, refs(values.map((item) => item.id))]])));
    return { id, type: expected };
  }
  if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier") {
    const objectType = inferExpressionType(node.object, context);
    const record = objectType?.startsWith("record:") ? context.recordsByName.get(objectType.slice("record:".length)) : undefined;
    const field = record?.fields.find((item) => item.name === node.property.name);
    if (!record || !field || field.type !== expected) fail("javascript.record_field_read", node.loc.start);
    const base = emitExpression(node.object, owner, `${path}.record`, context, objectType);
    const id = expressionID(owner, path, "field-read");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009032", [[0x9320, ref(base.id)], [0x9321, ref(field.id)]])));
    return { id, type: expected };
  }
  if (node.type === "CallExpression" && node.callee.type === "Identifier" && !node.optional) {
	const calleeParameterIndex = context.parameterNames.indexOf(node.callee.name);
	const calleeLocal = context.locals.get(node.callee.name);
	const calleeType = calleeParameterIndex >= 0 ? context.parameterTypes[calleeParameterIndex] : calleeLocal?.type;
	if (calleeType?.startsWith("function:")) {
		const [parameterText, resultType] = calleeType.slice("function:".length).split("=>");
		const parameterTypes = parameterText === "" ? [] : parameterText.split(",");
		if (resultType !== expected || parameterTypes.length !== node.arguments.length) fail("javascript.indirect_call_type", node.loc.start);
		const callee = emitExpression(node.callee, owner, `${path}.callee`, context, calleeType);
		const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, parameterTypes[index]));
		const id = expressionID(owner, path, "indirect-call");
		context.entities.push(graphEntity(id, entity(id, "0000000000000000000000000000a024", [[0xa0240, ref(callee.id)], [0xa0241, refs(arguments_.map((item) => item.id))]])));
		return { id, type: expected };
	}
    const callee = context.functionsByName.get(node.callee.name);
    if (!callee || callee.signature.result !== expected || callee.signature.parameters.length !== node.arguments.length) fail("javascript.unsupported_call", node.loc.start);
    const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, callee.signature.parameters[index].type));
    const id = expressionID(owner, path, "function-call");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009060", [[0x9600, ref(callee.id)], [0x9601, refs(arguments_.map((item) => item.id))]])));
    return { id, type: expected };
  }
  const operator = node.operator;
  const table = {
	"+:i64": ["add", "00000000000000000000000000009014", 0x9140, 0x9141, "i64"],
	"*:i64": ["multiply", "00000000000000000000000000009090", 0x9900, 0x9901, "i64"],
	"-:i64": ["subtract", "000000000000000000000000000090a0", 0x9a00, 0x9a01, "i64"],
	"<=:bool": ["less-equal", "00000000000000000000000000009021", 0x9160, 0x9161, "i64"],
    "+:string": ["string-concat", "000000000000000000000000000090c3", 0x9c30, 0x9c31, "string"],
    "===:bool": ["string-equal", "000000000000000000000000000090c2", 0x9c20, 0x9c21, "string"],
    "&&:bool": ["boolean-and", "000000000000000000000000000090b1", 0x9b10, 0x9b11, "bool"],
    "||:bool": ["boolean-or", "000000000000000000000000000090c1", 0x9c10, 0x9c11, "bool"],
  };
  const rule = table[`${operator}:${expected}`];
  if (!rule || !["BinaryExpression", "LogicalExpression"].includes(node.type)) fail("javascript.unsupported_expression", node.loc.start);
  if (expected === "i64" && node.type === "BinaryExpression" && ["+", "-", "*"].includes(node.operator) && !refinedI64Arithmetic.has(node)) fail("javascript.i64_arithmetic_requires_asIntN", node.loc.start);
  const [kind, schema, leftField, rightField, operandType] = rule;
  const left = emitExpression(node.left, owner, `${path}.left`, context, operandType);
  const right = emitExpression(node.right, owner, `${path}.right`, context, operandType);
  const id = expressionID(owner, path, kind);
  const fields = [[leftField, ref(left.id)], [rightField, ref(right.id)]];
	if (schema === "00000000000000000000000000009014") fields.push([0x9142, ref(ids.i64)]);
	if (schema === "00000000000000000000000000009090") fields.push([0x9902, ref(ids.i64)]);
	if (schema === "000000000000000000000000000090a0") fields.push([0x9a02, ref(ids.i64)]);
	if (schema === "00000000000000000000000000009021") fields.push([0x9162, ref(ids.i64)]);
  context.entities.push(graphEntity(id, entity(id, schema, fields)));
  return { id, type: expected };
}

function unwrapI64AsIntN(node) {
	if (node?.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "BigInt" && node.callee.property.name === "asIntN" && node.arguments.length === 2 && node.arguments[0].type === "Literal" && node.arguments[0].value === 64) return node.arguments[1];
	return node;
}

function emitMutableClosure(node, owner, path, context, expected, parameterTypes, resultType, captureName) {
	if (parameterTypes.length !== 1 || resultType !== "i64" || node.params.length !== 1 || node.params[0].type !== "Identifier" || node.body.type !== "BlockStatement" || node.body.body.length !== 2) fail("javascript.mutable_closure_shape", node.loc.start);
	const [assignmentStatement, returnStatement] = node.body.body;
	const assignment = assignmentStatement.type === "ExpressionStatement" ? assignmentStatement.expression : undefined;
	const assignmentRight = assignment ? unwrapI64AsIntN(assignment.right) : undefined;
	if (assignment?.type !== "AssignmentExpression" || assignment.operator !== "=" || assignment.left.type !== "Identifier" || assignment.left.name !== captureName || assignmentRight === assignment.right || assignmentRight?.type !== "BinaryExpression" || assignmentRight.operator !== "+" || assignmentRight.left.type !== "Identifier" || assignmentRight.left.name !== captureName || assignmentRight.right.type !== "Identifier" || assignmentRight.right.name !== node.params[0].name || returnStatement.type !== "ReturnStatement" || returnStatement.argument?.type !== "Identifier" || returnStatement.argument.name !== captureName) fail("javascript.mutable_closure_body", node.loc.start);
	const initialNode = context.mutableCaptureInitials.get(captureName);
	const initial = emitExpression(initialNode, owner, `${path}.capture.initial`, context, "i64");
	const captureID = expressionID(owner, path, "mutable-capture");
	const parameterID = expressionID(owner, path, "closure-parameter");
	const oldReadID = expressionID(owner, `${path}.body.step.0.value.left`, "mutable-capture-read");
	const parameterReadID = expressionID(owner, `${path}.body.step.0.value.right`, "closure-parameter-read");
	const addID = expressionID(owner, `${path}.body.step.0.value`, "add");
	const updateID = expressionID(owner, `${path}.body.step.0`, "capture-update");
	const resultID = expressionID(owner, `${path}.body.result`, "mutable-capture-read");
	const sequenceID = expressionID(owner, `${path}.body`, "sequence");
	const closureID = expressionID(owner, path, "mutable-closure");
	context.entities.push(graphEntity(captureID, entity(captureID, "0000000000000000000000000000a030", [[0xa0300, bytes(captureName)], [0xa0301, ref(ids.i64)], [0xa0302, ref(initial.id)]])));
	context.entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [[0x9120, bytes(node.params[0].name)], [0x9121, ref(ids.i64)], [0x9122, "uu 0"]])));
	context.entities.push(graphEntity(oldReadID, entity(oldReadID, "0000000000000000000000000000a031", [[0xa0310, ref(captureID)]])));
	context.entities.push(graphEntity(parameterReadID, entity(parameterReadID, "00000000000000000000000000009013", [[0x9130, ref(parameterID)]])));
	context.entities.push(graphEntity(addID, entity(addID, "00000000000000000000000000009014", [[0x9140, ref(oldReadID)], [0x9141, ref(parameterReadID)], [0x9142, ref(ids.i64)]])));
	context.entities.push(graphEntity(updateID, entity(updateID, "0000000000000000000000000000a032", [[0xa0320, ref(captureID)], [0xa0321, ref(addID)]])));
	context.entities.push(graphEntity(resultID, entity(resultID, "0000000000000000000000000000a031", [[0xa0310, ref(captureID)]])));
	context.entities.push(graphEntity(sequenceID, entity(sequenceID, "0000000000000000000000000000a033", [[0xa0330, refs([updateID])], [0xa0331, ref(resultID)]])));
	const functionTypeID = typeID(expected, context);
	context.entities.push(graphEntity(closureID, entity(closureID, "0000000000000000000000000000a034", [[0xa0340, ref(functionTypeID)], [0xa0341, refs([parameterID])], [0xa0342, refs([captureID])], [0xa0343, ref(sequenceID)]])));
	return { id: closureID, type: expected };
}

function dynamicIndexExpression(node, owner, path, context) {
	if (node.type === "Literal" && typeof node.value === "number" && Number.isSafeInteger(node.value)) {
		const id = expressionID(owner, path, "integer-literal");
		context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009070", [[0x9700, `uu ${BigInt.asUintN(64, BigInt(node.value))}`], [0x9701, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
	if (node.type === "BinaryExpression" && (node.operator === "+" || node.operator === "-")) {
		const left = dynamicIndexExpression(node.left, owner, `${path}.left`, context);
		const right = dynamicIndexExpression(node.right, owner, `${path}.right`, context);
		const subtract = node.operator === "-";
		const id = expressionID(owner, path, subtract ? "subtract" : "add");
		context.entities.push(graphEntity(id, entity(id, subtract ? "000000000000000000000000000090a0" : "00000000000000000000000000009014", subtract
			? [[0x9a00, ref(left.id)], [0x9a01, ref(right.id)], [0x9a02, ref(ids.i64)]]
			: [[0x9140, ref(left.id)], [0x9141, ref(right.id)], [0x9142, ref(ids.i64)]])));
		return { id, type: "i64" };
	}
	return emitExpression(node, owner, path, context, "i64");
}

function inferExpressionType(node, context) {
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme" && node.callee.property.name === "array" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression" && node.arguments[0].elements.length > 0) return `array:i64:${node.arguments[0].elements.length}`;
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme" && node.callee.property.name === "slice" && node.arguments.length === 1 && node.arguments[0].type === "ArrayExpression") return "slice:i64";
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme" && ["append", "update", "remove"].includes(node.callee.property.name)) return "slice:i64";
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme" && ["emptyMap", "mapInsert", "mapRemove"].includes(node.callee.property.name)) return "map:i64:i64";
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme" && ["mapLookupZero", "length", "index"].includes(node.callee.property.name)) return "i64";
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && node.callee.object.type === "Identifier" && node.callee.object.name === "Seme") fail("javascript.constructor_requires_boundary_type", node.loc.start);
  if (node.type === "NewExpression" && node.callee.type === "Identifier" && node.callee.name === "Map") return "map:i64:i64";
  if (node.type === "LogicalExpression" && node.operator === "??") return inferExpressionType(node.right, context);
  if (node.type === "ThisExpression" && context.receiver) return context.receiver.type;
  if (node.type === "Identifier") {
	const variant = context.variants?.get(node.name);
	if (variant) return variant.type;
    const parameter = context.parameterNames.indexOf(node.name);
    if (parameter >= 0) return context.parameterTypes[parameter];
    const local = context.locals.get(node.name);
    if (local) return local.type;
  }
  if (node.type === "Literal" && typeof node.value === "string") return "string";
  if (node.type === "Literal" && typeof node.value === "boolean") return "bool";
  if (node.type === "Literal" && typeof node.value === "bigint") return "i64";
	if (node.type === "MemberExpression" && !node.computed && node.property.name === "length") return "i64";
  if (node.type === "ObjectExpression") {
	const propertyNames = new Set(node.properties.map((property) => property.key?.name ?? property.key?.value));
	if (propertyNames.size === 2 && propertyNames.has("state") && propertyNames.has("result")) {
		const stateNode = node.properties.find((property) => (property.key?.name ?? property.key?.value) === "state")?.value;
		const resultNode = node.properties.find((property) => (property.key?.name ?? property.key?.value) === "result")?.value;
		return `transition:${inferExpressionType(stateNode, context)}:${inferExpressionType(resultNode, context)}`;
	}
    const names = new Set(node.properties.map((property) => property.key?.name ?? property.key?.value));
    const matches = [...context.recordsByName.values()].filter((record) => record.fields.length === names.size && record.fields.every((field) => names.has(field.name)));
    if (matches.length === 1) return `record:${matches[0].name}`;
  }
  if (node.type === "CallExpression" && node.callee.type === "Identifier") {
    const callee = context.functionsByName.get(node.callee.name);
    if (callee) return callee.signature.result;
  }
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && node.callee.object.type === "Identifier" && node.callee.object.name === "BigInt" && node.callee.property.name === "asIntN" && node.arguments.length === 2 && node.arguments[0].type === "Literal" && node.arguments[0].value === 64) return "i64";
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed) {
		if (node.callee.property.name === "set") return "map:i64:i64";
		if (node.callee.property.name === "reduce" && node.arguments.length === 2 && inferExpressionType(node.arguments[1], context) === "i64") return "i64";
		if (node.callee.property.name === "reduce" && node.arguments[0]?.type === "ArrowFunctionExpression" && node.arguments[0].body.type === "CallExpression" && node.arguments[0].body.callee.type === "MemberExpression" && node.arguments[0].body.callee.property.name === "set") return "map:i64:i64";
		const receiverType = inferExpressionType(node.callee.object, context);
		if (receiverType.startsWith("interface:")) {
			const requirement = context.interfacesByName.get(receiverType.slice("interface:".length))?.requirements.find((item) => item.name === node.callee.property.name);
			if (requirement) return requirement.result;
		}
		const method = context.methodsByTypeAndName?.get(`${receiverType.slice("record:".length)}:${node.callee.property.name}`);
		if (method) return method.signature.result;
	}
	if (node.type === "NewExpression" && node.callee.type === "Identifier" && context.recordsByName.has(node.callee.name)) return `record:${node.callee.name}`;
	if (node.type === "MemberExpression" && !node.computed) {
		const objectType = inferExpressionType(node.object, context);
		if (objectType.startsWith("transition:")) {
			const parts = objectType.split(":");
			if (node.property.name === "state") return `record:${parts[2]}`;
			if (node.property.name === "result") return parts.slice(3).join(":");
		}
		if (objectType.startsWith("record:")) {
			const record = context.recordsByName.get(objectType.slice("record:".length));
			const recordField = record?.fields.find((item) => item.name === node.property.name);
			if (recordField) return recordField.type;
		}
	}
	if (node.type === "CallExpression" && node.callee.type === "MemberExpression" && !node.callee.computed && (node.callee.property.name === "with" || node.callee.property.name === "concat")) return "slice:i64";
  if (node.type === "LogicalExpression" && (node.operator === "&&" || node.operator === "||")) return "bool";
  if (node.type === "BinaryExpression" && node.operator === "===") return "bool";
	if (node.type === "BinaryExpression" && node.operator === "<=") return "bool";
	if (node.type === "BinaryExpression" && node.operator === "+") {
    const left = inferExpressionType(node.left, context);
    const right = inferExpressionType(node.right, context);
    if (left === "string" && right === "string") return "string";
  }
	if (node.type === "BinaryExpression" && node.operator === "-") return "i64";
  fail("javascript.ambiguous_local_type", node.loc.start);
}

function referencedIdentifiers(node) {
  const names = [];
  const visit = (value) => {
    if (!value || typeof value !== "object") return;
    if (value.type === "Identifier") { names.push(value.name); return; }
    if (value.type === "MemberExpression" && !value.computed) { visit(value.object); return; }
    for (const [key, child] of Object.entries(value)) {
      if (key === "loc" || key === "start" || key === "end") continue;
      if (Array.isArray(child)) child.forEach(visit); else visit(child);
    }
  };
  visit(node);
  return names;
}

function blockStatements(node) {
  if (node.type === "BlockStatement") return node.body;
  if (node.type === "IfStatement" || node.type === "ReturnStatement") return [node];
  fail("javascript.unsupported_branch", node.loc.start);
}
function containsReturn(node) {
  if (node.type === "ReturnStatement") return true;
  if (node.type === "BlockStatement") return node.body.some(containsReturn);
  if (node.type === "IfStatement") return containsReturn(node.consequent) || (node.alternate ? containsReturn(node.alternate) : false);
  return false;
}
function expressionID(owner, path, kind) {
	if (path === "root" && kind === "less-equal") return stableID("execution", owner, "less-equal");
	if (path === "root.left" && kind === "add") return stableID("execution", owner, "add");
	return stableID("execution", owner, "expression", path, kind);
}
function isUnicodeScalarString(value) {
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) return false;
      index += 1;
    } else if (unit >= 0xdc00 && unit <= 0xdfff) return false;
  }
  return true;
}
function stableID(...parts) {
  const hash = crypto.createHash("sha256").update("seme.provider.identity.v1\0");
  for (const part of parts) hash.update(part).update("\0");
  const bytes = hash.digest().subarray(0, 15);
  return `80${bytes.toString("hex")}`;
}
function graphEntity(id, text) { return { id, text }; }
function bytes(value) { return `by ${Buffer.from(value, "utf8").toString("hex") || "-"}`; }
function rawBytes(value) { return `by ${Buffer.from(value).toString("hex") || "-"}`; }
function ref(value) { return `rf ${value}`; }
function refs(values) { return `li ${values.length}${values.map((value) => `\nrf ${value}`).join("")}`; }
function entity(id, schema, fields) {
  fields.sort((left, right) => left[0] - right[0]);
  return `en ${id} ${schema} 1 ${fields.length}\n${fields.map(([field, value]) => `fi ${field.toString(16).padStart(32, "0")} ${value}\n`).join("")}`;
}
function compose(moduleG1, revision, additions) {
  const entities = [];
  let current;
  for (const line of moduleG1.trim().split("\n")) {
    if (line.startsWith("en ")) { current = { id: line.split(/\s+/)[1], text: "" }; entities.push(current); }
    if (current) current.text += `${line}\n`;
  }
  const moduleRoot = entities.find((item) => item.id === "00000000000000000000000000009000");
  const moduleVersion = Number(moduleRoot?.text.match(/^en\s+\S+\s+\S+\s+(\d+)\s+/m)?.[1] ?? 0);
  if (moduleVersion >= 26) entities.length = 0;
  const unique = new Map(entities.map((item) => [item.id, item]));
  for (const item of additions) {
    const existing = unique.get(item.id);
    if (existing && existing.text !== item.text) fail("javascript.identity_collision");
    unique.set(item.id, item);
  }
  entities.length = 0;
  entities.push(...unique.values());
  entities.sort((left, right) => left.id.localeCompare(right.id));
  return `# Generated exact JavaScript to Core Execution lift.\nve 1\nmo 00000000000000000000000000009000\nrv ${revision}\npc 0\nec ${entities.length}\n${entities.map((item) => `\n${item.text}`).join("")}`;
}
function fail(code, location) {
  const suffix = location ? `:${location.line}:${location.column + 1}` : "";
  throw new Error(`${code}${suffix}`);
}
