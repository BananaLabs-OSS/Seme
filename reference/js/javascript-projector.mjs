const schema = {
  function: "00000000000000000000000000009011",
  parameter: "00000000000000000000000000009012",
  read: "00000000000000000000000000009013",
  program: "00000000000000000000000000009015",
  boolType: "00000000000000000000000000009020",
  stringType: "00000000000000000000000000009040",
  stringLiteral: "00000000000000000000000000009050",
  block: "00000000000000000000000000009080",
  returned: "00000000000000000000000000009081",
  boolLiteral: "000000000000000000000000000090b0",
  boolAnd: "000000000000000000000000000090b1",
  branch: "000000000000000000000000000090c0",
  boolOr: "000000000000000000000000000090c1",
  stringEqual: "000000000000000000000000000090c2",
  stringConcat: "000000000000000000000000000090c3",
};

export function projectJavaScript(canonicalG1) {
  const graph = parseG1(canonicalG1);
  const programs = [...graph.values()].filter((entity) => entity.schema === schema.program);
  if (programs.length !== 1) fail("javascript_projection.requires_one_program");
  const entryID = reference(field(programs[0], 0x9151));
  const fn = required(graph, entryID, schema.function);
  const name = text(field(fn, 0x9110));
  if (!/^[A-Za-z_$][\w$]*$/.test(name)) fail("javascript_projection.invalid_function_name");
  const parameterIDs = references(field(fn, 0x9111));
  const parameters = parameterIDs.map((id) => {
    const parameter = required(graph, id, schema.parameter);
    const parameterName = text(field(parameter, 0x9120));
    if (!/^[A-Za-z_$][\w$]*$/.test(parameterName)) fail("javascript_projection.invalid_parameter_name");
    return { id, name: parameterName, type: typeName(reference(field(parameter, 0x9121)), graph) };
  });
  const result = typeName(reference(field(fn, 0x9112)), graph);
  const context = { graph, parameters: new Map(parameters.map((item) => [item.id, item])) };
  const body = projectBlock(reference(field(fn, 0x9113)), context, "  ");
  const jsdoc = ["/**", ...parameters.map((item) => ` * @param {${item.type}} ${item.name}`), ` * @returns {${result}}`, " */"];
  return `${jsdoc.join("\n")}\nexport function ${name}(${parameters.map((item) => item.name).join(", ")}) {\n${body}\n}\n`;
}

function projectBlock(id, context, indent) {
  const block = required(context.graph, id, schema.block);
  const statements = references(field(block, 0x9800));
  if (statements.length !== 1) fail("javascript_projection.block_arity");
  const statement = required(context.graph, statements[0]);
  if (statement.schema === schema.returned) {
    const values = references(field(statement, 0x9810));
    if (values.length !== 1) fail("javascript_projection.return_arity");
    return `${indent}return ${projectExpression(values[0], context)};`;
  }
  if (statement.schema === schema.branch) {
    const condition = projectExpression(reference(field(statement, 0x9c00)), context);
    const thenBody = projectBlock(reference(field(statement, 0x9c01)), context, `${indent}  `);
    const elseBody = projectBlock(reference(field(statement, 0x9c02)), context, `${indent}  `);
    return `${indent}if (${condition}) {\n${thenBody}\n${indent}} else {\n${elseBody}\n${indent}}`;
  }
  fail("javascript_projection.unsupported_statement");
}

function projectExpression(id, context) {
  const expression = required(context.graph, id);
  if (expression.schema === schema.read) {
    const parameter = context.parameters.get(reference(field(expression, 0x9130)));
    if (!parameter) fail("javascript_projection.unknown_parameter");
    return parameter.name;
  }
  if (expression.schema === schema.stringLiteral) return JSON.stringify(text(field(expression, 0x9500)));
  if (expression.schema === schema.boolLiteral) return atom(field(expression, 0x9b00)) === "tr" ? "true" : "false";
  const binary = new Map([
    [schema.boolAnd, [0x9b10, 0x9b11, "&&"]],
    [schema.boolOr, [0x9c10, 0x9c11, "||"]],
    [schema.stringEqual, [0x9c20, 0x9c21, "==="]],
    [schema.stringConcat, [0x9c30, 0x9c31, "+"]],
  ]).get(expression.schema);
  if (!binary) fail("javascript_projection.unsupported_expression");
  const [leftField, rightField, operator] = binary;
  return `(${projectExpression(reference(field(expression, leftField)), context)} ${operator} ${projectExpression(reference(field(expression, rightField)), context)})`;
}

function typeName(id, graph) {
  const type = required(graph, id);
  if (type.schema === schema.stringType) return "string";
  if (type.schema === schema.boolType) return "boolean";
  fail("javascript_projection.unsupported_type");
}

function parseG1(source) {
  const lines = source.trim().split("\n");
  const graph = new Map();
  for (let index = 0; index < lines.length;) {
    if (!lines[index].startsWith("en ")) { index += 1; continue; }
    const header = lines[index].trim().split(/\s+/);
    const entity = { id: header[1], schema: header[2], fields: new Map() };
    index += 1;
    for (let count = 0; count < Number(header[4]); count += 1) {
      const match = /^fi ([0-9a-f]{32}) (.*)$/.exec(lines[index]);
      if (!match) fail("javascript_projection.malformed_field");
      const value = [match[2]];
      const list = /^li (\d+)$/.exec(match[2]);
      index += 1;
      if (list) {
        for (let item = 0; item < Number(list[1]); item += 1, index += 1) value.push(lines[index].trim());
      }
      const fieldID = BigInt(`0x${match[1]}`);
      if (entity.fields.has(fieldID)) fail("javascript_projection.duplicate_field");
      entity.fields.set(fieldID, value);
    }
    if (graph.has(entity.id)) fail("javascript_projection.duplicate_entity");
    graph.set(entity.id, entity);
  }
  return graph;
}
function required(graph, id, expected) {
  const entity = graph.get(id);
  if (!entity || (expected && entity.schema !== expected)) fail("javascript_projection.missing_or_mistyped_entity");
  return entity;
}
function field(entity, id) {
  const value = entity.fields.get(BigInt(id));
  if (!value) fail("javascript_projection.missing_field");
  return value;
}
function atom(value) { return value[0]; }
function reference(value) {
  const match = /^rf ([0-9a-f]{32})$/.exec(atom(value));
  if (!match) fail("javascript_projection.invalid_reference");
  return match[1];
}
function references(value) {
  const match = /^li (\d+)$/.exec(atom(value));
  if (!match || value.length !== Number(match[1]) + 1) fail("javascript_projection.invalid_reference_list");
  return value.slice(1).map((item) => reference([item]));
}
function text(value) {
  const match = /^by ([-0-9a-f]*)$/.exec(atom(value));
  if (!match) fail("javascript_projection.invalid_bytes");
  if (match[1] === "-") return "";
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(Buffer.from(match[1], "hex"));
  } catch {
    fail("javascript_projection.invalid_utf8");
  }
}
function fail(code) { throw new Error(code); }
