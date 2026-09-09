const schema = {
  boolType: "00000000000000000000000000009020",
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  program: "00000000000000000000000000009015",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  call: "00000000000000000000000000009060",
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
  const boolTypes = [...graph.values()].filter((item) => item.schema === schema.boolType);
  if (boolTypes.length !== 1) fail("lua_projection.requires_bool_type");
  const context = { graph, names, boolTypeID: boolTypes[0].id };
  return functionIDs.map((id) => projectFunction(id, id === entryID, context)).join("\n\n") + "\n";
}

function projectFunction(id, exported, context) {
  const fn = required(context.graph, id, schema.function);
  const parameterIDs = references(field(fn, 0x9111));
  const parameters = parameterIDs.map((parameterID) => {
    const parameter = required(context.graph, parameterID, schema.parameter);
    const name = text(field(parameter, 0x9120));
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) fail("lua_projection.invalid_parameter_name");
    if (reference(field(parameter, 0x9121)) !== context.boolTypeID) fail("lua_projection.unsupported_type");
    return { id: parameterID, name };
  });
  if (reference(field(fn, 0x9112)) !== context.boolTypeID) fail("lua_projection.unsupported_type");
  const block = required(context.graph, reference(field(fn, 0x9113)), schema.block);
  const statements = references(field(block, 0x9800));
  if (statements.length !== 1) fail("lua_projection.block_profile");
  const returned = required(context.graph, statements[0], schema.returned);
  const local = { ...context, parameters: new Map(parameters.map((item) => [item.id, item.name])) };
  const returnedValues = references(field(returned, 0x9810));
  if (returnedValues.length !== 1) fail("lua_projection.return_arity");
  const expression = projectExpression(returnedValues[0], local);
  return `${parameters.map((item) => `---@param ${item.name} boolean`).join("\n")}${parameters.length ? "\n" : ""}---@return boolean\n${exported ? "" : "local "}function ${context.names.get(id)}(${parameters.map((item) => item.name).join(", ")})\n  return ${expression}\nend`;
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.read) {
    const name = context.parameters.get(reference(field(expression, 0x9130)));
    if (!name) fail("lua_projection.read_scope");
    return name;
  }
  if (expression.schema === schema.call) {
    const callee = context.names.get(reference(field(expression, 0x9600)));
    if (!callee) fail("lua_projection.call_membership");
    return `${callee}(${references(field(expression, 0x9601)).map((argument) => projectExpression(argument, context)).join(", ")})`;
  }
  fail("lua_projection.unsupported_expression");
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
      const match = /^fi\s+([0-9a-f]{32})\s+(by|rf|li|uu)\s+(.+)$/.exec(lines[index]);
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
      } else item.fields.set(BigInt(`0x${match[1]}`), { kind: match[2], value: match[3] });
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
function fail(code) { throw new Error(code); }
