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
  variantBinding: "0000000000000000000000000000a060",
  variantRead: "0000000000000000000000000000a061",
  resultMatch: "0000000000000000000000000000a062",
  optionMatch: "0000000000000000000000000000a063",
  bytesLiteral: "0000000000000000000000000000a064",
  bytesEqual: "0000000000000000000000000000a065",
  stringLiteral: "00000000000000000000000000009050",
  stringEqual: "000000000000000000000000000090c2",
  boolLiteral: "000000000000000000000000000090b0",
};

export function projectLua(canonicalG1) {
  const graph = parseG1(canonicalG1);
  const programs = [...graph.values()].filter((item) => item.schema === schema.program);
  if (programs.length !== 1) fail("lua_projection.requires_one_program");
  const functionIDs = references(field(programs[0], 0x9150));
  const entryID = reference(field(programs[0], 0x9151));
  if (!functionIDs.includes(entryID)) fail("lua_projection.entry_membership");
  const names = new Map();
  for (const id of functionIDs) {
    const name = text(field(required(graph, id, schema.function), 0x9110));
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) || [...names.values()].includes(name)) fail("lua_projection.invalid_function_name");
    names.set(id, name);
  }
  const types = new Map([...graph.values()].flatMap((item) => {
    const name = ({ [schema.i64Type]: "seme.i64", [schema.boolType]: "boolean", [schema.stringType]: "seme.text", [schema.bytesType]: "seme.bytes" })[item.schema];
    return name ? [[item.id, name]] : [];
  }));
  const resolveType = (id, visiting = new Set()) => {
    if (types.has(id)) return types.get(id);
    if (visiting.has(id)) fail("lua_projection.recursive_type");
    visiting.add(id);
    const item = required(graph, id);
    let name;
    if (item.schema === schema.optionType) name = `seme.option<${resolveType(reference(field(item, 0xa0500)), visiting)}>`;
    if (item.schema === schema.resultType) name = `seme.result<${resolveType(reference(field(item, 0x9400)), visiting)},${resolveType(reference(field(item, 0x9401)), visiting)}>`;
    if (item.schema === schema.fixedArrayType) name = `seme.array<${resolveType(reference(field(item, 0x9f20)), visiting)},${unsigned(field(item, 0x9f21))}>`;
    if (item.schema === schema.sliceType) name = `seme.slice<${resolveType(reference(field(item, 0x9f80)), visiting)}>`;
    if (item.schema === schema.mapType) name = `seme.map<${resolveType(reference(field(item, 0xa0400)), visiting)},${resolveType(reference(field(item, 0xa0401)), visiting)}>`;
    if (item.schema === schema.recordType) name = text(field(item, 0x9300));
    if (!name) fail("lua_projection.unsupported_type");
    types.set(id, name); visiting.delete(id); return name;
  };
  for (const id of functionIDs) {
    const fn = required(graph, id, schema.function);
    resolveType(reference(field(fn, 0x9112)));
    for (const parameter of references(field(fn, 0x9111))) resolveType(reference(field(required(graph, parameter, schema.parameter), 0x9121)));
  }
  const context = { graph, names, types };
  const records = [...graph.values()].filter((item) => item.schema === schema.recordType).map((record) => {
    const name = text(field(record, 0x9300));
    const fields = references(field(record, 0x9301)).map((id) => { const item = required(graph, id, schema.recordField); return `---@field ${text(field(item, 0x9310))} ${resolveType(reference(field(item, 0x9311)))}`; });
    return `---@class ${name}\n${fields.join("\n")}`;
  });
  return [...records, ...functionIDs.map((id) => projectFunction(id, id === entryID, context))].join("\n\n") + "\n";
}

function projectFunction(id, exported, context) {
  const fn = required(context.graph, id, schema.function);
  const parameterIDs = references(field(fn, 0x9111));
  const parameters = parameterIDs.map((parameterID) => {
    const parameter = required(context.graph, parameterID, schema.parameter);
    const name = text(field(parameter, 0x9120));
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) fail("lua_projection.invalid_parameter_name");
    const type = context.types.get(reference(field(parameter, 0x9121)));
    if (!type) fail("lua_projection.unsupported_type");
    return { id: parameterID, name, type };
  });
  const resultType = context.types.get(reference(field(fn, 0x9112)));
  if (!resultType) fail("lua_projection.unsupported_type");
  const block = required(context.graph, reference(field(fn, 0x9113)), schema.block);
  const statements = references(field(block, 0x9800));
  if (statements.length !== 1) fail("lua_projection.block_profile");
  const returned = required(context.graph, statements[0], schema.returned);
  const local = { ...context, parameters: new Map(parameters.map((item) => [item.id, item])) };
  const returnedValues = references(field(returned, 0x9810));
  if (returnedValues.length !== 1) fail("lua_projection.return_arity");
  const expression = projectExpression(returnedValues[0], local);
  return `${parameters.map((item) => `---@param ${item.name} ${item.type}`).join("\n")}${parameters.length ? "\n" : ""}---@return ${resultType}\n${exported ? "" : "local "}function ${context.names.get(id)}(${parameters.map((item) => item.name).join(", ")})\n  return ${expression}\nend`;
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.read) {
    const parameter = context.parameters.get(reference(field(expression, 0x9130)));
    if (!parameter) fail("lua_projection.read_scope");
    return parameter.name;
  }
  if (expression.schema === schema.call) {
    const callee = context.names.get(reference(field(expression, 0x9600)));
    if (!callee) fail("lua_projection.call_membership");
    return `${callee}(${references(field(expression, 0x9601)).map((argument) => projectExpression(argument, context)).join(", ")})`;
  }
  if (expression.schema === schema.optionNone) return "Seme.none()";
  if (expression.schema === schema.optionSome) return `Seme.some(${projectExpression(reference(field(expression, 0xa0521)), context)})`;
  if (expression.schema === schema.resultOk) return `Seme.ok(${projectExpression(reference(field(expression, 0x9411)), context)})`;
  if (expression.schema === schema.resultError) return `Seme.err(${projectExpression(reference(field(expression, 0x9421)), context)})`;
  if (expression.schema === schema.fieldRead) {
    const member = required(context.graph, reference(field(expression, 0x9321)), schema.recordField);
    return `Seme.field(${projectExpression(reference(field(expression, 0x9320)), context)}, "${text(field(member, 0x9310))}")`;
  }
  if (expression.schema === schema.fixedArrayConstruct) return `Seme.array(${references(field(expression, 0x9f31)).map((value) => projectExpression(value, context)).join(", ")})`;
  if (expression.schema === schema.collectionLength) return `Seme.length(${projectExpression(reference(field(expression, 0x9f90)), context)})`;
  if (expression.schema === schema.indexRead) {
    const index = required(context.graph, reference(field(expression, 0x9f41)), schema.integerLiteral);
    return `Seme.index_zero(${projectExpression(reference(field(expression, 0x9f40)), context)}, ${unsigned(field(index, 0x9700))})`;
  }
  if (expression.schema === schema.emptyMap) return `Seme.empty_map("${mapValueDescriptor(reference(field(expression, 0xa0410)), context)}")`;
  if (expression.schema === schema.mapLookup) {
    const mapID = reference(field(expression, 0xa0420));
    return `Seme.lookup_zero(${projectExpression(mapID, context)}, ${projectExpression(reference(field(expression, 0xa0421)), context)}, "${mapExpressionDescriptor(mapID, context)}")`;
  }
  if (expression.schema === schema.mapUpdate) return `Seme.map_update(${projectExpression(reference(field(expression, 0xa0430)), context)}, ${projectExpression(reference(field(expression, 0xa0431)), context)}, ${projectExpression(reference(field(expression, 0xa0432)), context)})`;
  if (expression.schema === schema.variantRead) return bindingName(reference(field(expression, 0xa0610)), context);
  if (expression.schema === schema.bytesLiteral) return `Seme.bytes_literal(${JSON.stringify(text(field(expression, 0xa0640)))})`;
  if (expression.schema === schema.bytesEqual) return `Seme.bytes_equal(${projectExpression(reference(field(expression, 0xa0650)), context)}, ${projectExpression(reference(field(expression, 0xa0651)), context)})`;
  if (expression.schema === schema.stringLiteral) return JSON.stringify(text(field(expression, 0x9500)));
  if (expression.schema === schema.stringEqual) return `Seme.text_equal(${projectExpression(reference(field(expression, 0x9c20)), context)}, ${projectExpression(reference(field(expression, 0x9c21)), context)})`;
  if (expression.schema === schema.boolLiteral) return boolean(field(expression, 0x9b00)) ? "true" : "false";
  if (expression.schema === schema.resultMatch) {
    const ok = reference(field(expression, 0xa0621)), error = reference(field(expression, 0xa0623));
    return `Seme.match_result(${projectExpression(reference(field(expression, 0xa0620)), context)}, function(${bindingName(ok, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0622)), context)} end, function(${bindingName(error, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0624)), context)} end)`;
  }
  if (expression.schema === schema.optionMatch) {
    const some = reference(field(expression, 0xa0632));
    const none = projectBlockExpression(reference(field(expression, 0xa0631)), context);
    if (none !== "false") fail("lua_projection.option_none_profile");
    return `Seme.match_option(${projectExpression(reference(field(expression, 0xa0630)), context)}, false, function(${bindingName(some, context)}) return ${projectBlockExpression(reference(field(expression, 0xa0633)), context)} end)`;
  }
  fail("lua_projection.unsupported_expression");
}

function projectBlockExpression(id, context) {
  const statements = references(field(required(context.graph, id, schema.block), 0x9800));
  if (statements.length !== 1) fail("lua_projection.match_block");
  const values = references(field(required(context.graph, statements[0], schema.returned), 0x9810));
  if (values.length !== 1) fail("lua_projection.match_return");
  return projectExpression(values[0], context);
}
function bindingName(id, context) {
  const value = text(field(required(context.graph, id, schema.variantBinding), 0xa0600));
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)) fail("lua_projection.binding_name");
  return value;
}

function parseG1(source) {
  const graph = new Map();
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  for (let index = 0; index < lines.length;) {
    if (!lines[index].startsWith("en ")) { index += 1; continue; }
    const header = lines[index].split(/\s+/);
    if (header.length !== 5) fail("lua_projection.invalid_entity");
    const item = { id: header[1], schema: header[2], fields: new Map() };
    const count = Number(header[4]); index += 1;
    for (let seen = 0; seen < count; seen += 1, index += 1) {
      const match = /^fi\s+([0-9a-f]{32})\s+(by|rf|li|uu|tr|fa)(?:\s+(.+))?$/.exec(lines[index]);
      if (!match) fail("lua_projection.invalid_field");
      if (match[2] === "li") {
        const length = Number(match[3]); const values = [];
        for (let offset = 0; offset < length; offset += 1) {
          index += 1;
          const member = /^rf\s+([0-9a-f]{32})$/.exec(lines[index]);
          if (!member) fail("lua_projection.invalid_list");
          values.push(member[1]);
        }
        item.fields.set(BigInt(`0x${match[1]}`), { kind: "li", value: values });
      } else item.fields.set(BigInt(`0x${match[1]}`), { kind: match[2], value: match[3] ?? "" });
    }
    if (graph.has(item.id)) fail("lua_projection.duplicate_entity");
    graph.set(item.id, item);
  }
  return graph;
}
function required(graph, id, expected) { const value = graph.get(id); if (!value || expected && value.schema !== expected) fail("lua_projection.invalid_reference"); return value; }
function field(entity, key) { const value = entity.fields.get(BigInt(key)); if (!value) fail("lua_projection.missing_field"); return value; }
function reference(value) { if (value.kind !== "rf") fail("lua_projection.expected_reference"); return value.value; }
function references(value) { if (value.kind !== "li") fail("lua_projection.expected_references"); return value.value; }
function text(value) { if (value.kind !== "by") fail("lua_projection.expected_text"); if (value.value === "-") return ""; const data = Buffer.from(value.value, "hex"); const decoded = new TextDecoder("utf-8", { fatal: true }).decode(data); return decoded; }
function unsigned(value) { if (value.kind !== "uu" || !/^(?:0|[1-9][0-9]*)$/.test(value.value)) fail("lua_projection.expected_unsigned"); return value.value; }
function boolean(value) { if (value.kind === "tr") return true; if (value.kind === "fa") return false; fail("lua_projection.expected_boolean"); }
function mapValueDescriptor(typeID, context) {
  const map = required(context.graph, typeID, schema.mapType);
  const type = context.types.get(reference(field(map, 0xa0401)));
  if (type !== "seme.i64" && type !== "seme.text" && type !== "seme.bytes") fail("lua_projection.map_zero_unsupported");
  return type.slice("seme.".length);
}
function mapExpressionDescriptor(expressionID, context) {
  const expression = required(context.graph, expressionID, schema.read);
  const parameter = required(context.graph, reference(field(expression, 0x9130)), schema.parameter);
  return mapValueDescriptor(reference(field(parameter, 0x9121)), context);
}
function fail(code) { throw new Error(code); }
