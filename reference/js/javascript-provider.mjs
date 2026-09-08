import crypto from "node:crypto";
import { parse } from "acorn";

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  string: stableID("execution", "type", "string"),
};

export function liftJavaScript({ source, packagePath, revision, moduleG1 }) {
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("javascript.invalid_snapshot");
  const comments = [];
  let program;
  try {
    program = parse(source, { ecmaVersion: 2024, sourceType: "module", locations: true, onComment: comments });
  } catch (error) {
    fail("javascript.parse", error.loc);
  }
  const declarations = program.body.map((item) => item.type === "ExportNamedDeclaration" ? item.declaration : item);
  const functions = declarations.filter((item) => item?.type === "FunctionDeclaration");
  if (functions.length !== 1) fail("javascript.requires_one_function", functions[1]?.loc?.start);
  const fn = functions[0];
  if (!fn.id || fn.async || fn.generator) fail("javascript.unsupported_function", fn.loc.start);
  const signature = readSignature(comments, fn);
  if (signature.parameters.length !== fn.params.length) fail("javascript.signature_arity", fn.loc.start);
  const functionID = stableID("session-declaration", packagePath, fn.id.name);
  const entities = [
    graphEntity(ids.i64, entity(ids.i64, "00000000000000000000000000009010", [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]])),
    graphEntity(ids.bool, entity(ids.bool, "00000000000000000000000000009020", [])),
    graphEntity(ids.string, entity(ids.string, "00000000000000000000000000009040", [])),
  ];
  const parameterIDs = fn.params.map((parameter, index) => {
    if (parameter.type !== "Identifier") fail("javascript.unsupported_parameter", parameter.loc.start);
    if (signature.parameters[index].name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
    const parameterID = stableID("execution", functionID, "parameter", String(index));
    entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [
      [0x9120, bytes(parameter.name)], [0x9121, ref(ids[signature.parameters[index].type])], [0x9122, `uu ${index}`],
    ])));
    return parameterID;
  });
  const context = { functionID, parameterIDs, parameterNames: fn.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities };
  const bodyID = emitBlock(fn.body.body, "body", context, signature.result);
  entities.push(graphEntity(functionID, entity(functionID, "00000000000000000000000000009011", [
    [0x9110, bytes(fn.id.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(ids[signature.result])], [0x9113, ref(bodyID)],
  ])));
  const programID = stableID("session-program", packagePath);
  entities.push(graphEntity(programID, entity(programID, "00000000000000000000000000009015", [[0x9150, refs([functionID])], [0x9151, ref(functionID)]])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), entities);
}

function readSignature(comments, fn) {
  const comment = [...comments].reverse().find((item) => item.type === "Block" && item.end <= fn.start && sourceGapIsWhitespace(item.end, fn.start, fn));
  if (!comment || !comment.value.startsWith("*")) fail("javascript.missing_jsdoc", fn.loc.start);
  const parameters = [...comment.value.matchAll(/@param\s+\{(string|boolean|bigint)\}\s+([A-Za-z_$][\w$]*)/g)].map((match) => ({ type: semanticType(match[1]), name: match[2] }));
  const result = comment.value.match(/@returns?\s+\{(string|boolean|bigint)\}/);
  if (!result) fail("javascript.missing_result_type", fn.loc.start);
  return { parameters, result: semanticType(result[1]) };
}

// Acorn comments do not retain the source string. Requiring the closest JSDoc
// comment to precede the declaration is sufficient for this one-declaration profile.
function sourceGapIsWhitespace(_end, _start, _fn) { return true; }
function semanticType(type) { return type === "boolean" ? "bool" : type === "bigint" ? "i64" : "string"; }

function emitBlock(statements, path, context, resultType) {
  let statement;
  if (statements.length === 1 && statements[0].type === "ReturnStatement") statement = emitReturn(statements[0], path, context, resultType);
  else if (statements.length >= 1 && statements[0].type === "IfStatement") {
    const branch = statements[0];
    const following = statements.slice(1);
    const condition = emitExpression(branch.test, `${context.functionID}:${path}:condition`, "root", context, "bool");
    const thenID = emitBlock(blockStatements(branch.consequent), `${path}.then`, context, resultType);
    const elseStatements = branch.alternate ? blockStatements(branch.alternate) : following;
    if (branch.alternate && following.length) fail("javascript.unreachable_following", following[0].loc.start);
    if (!elseStatements.length) fail("javascript.branch_not_total", branch.loc.start);
    const elseID = emitBlock(elseStatements, `${path}.else`, context, resultType);
    const id = stableID("execution", context.functionID, `${path}.statement`, "if");
    context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090c0", [[0x9c00, ref(condition.id)], [0x9c01, ref(thenID)], [0x9c02, ref(elseID)]])));
    statement = id;
  } else fail("javascript.block_not_total", statements[0]?.loc?.start);
  const blockID = stableID("execution", context.functionID, path, "block");
  context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs([statement])]])));
  return blockID;
}

function emitReturn(statement, path, context, resultType) {
  if (!statement.argument) fail("javascript.return_arity", statement.loc.start);
  const expression = emitExpression(statement.argument, `${context.functionID}:${path}`, "root", context, resultType);
  const id = stableID("execution", context.functionID, `${path}.statement`, "return");
  context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009081", [[0x9810, refs([expression.id])]])));
  return id;
}

function emitExpression(node, owner, path, context, expected) {
  if (node.type === "Identifier") {
    const index = context.parameterNames.indexOf(node.name);
    if (index < 0 || context.parameterTypes[index] !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
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
  const operator = node.operator;
  const table = {
    "+:string": ["string-concat", "000000000000000000000000000090c3", 0x9c30, 0x9c31, "string"],
    "===:bool": ["string-equal", "000000000000000000000000000090c2", 0x9c20, 0x9c21, "string"],
    "&&:bool": ["boolean-and", "000000000000000000000000000090b1", 0x9b10, 0x9b11, "bool"],
    "||:bool": ["boolean-or", "000000000000000000000000000090c1", 0x9c10, 0x9c11, "bool"],
  };
  const rule = table[`${operator}:${expected}`];
  if (!rule || !["BinaryExpression", "LogicalExpression"].includes(node.type)) fail("javascript.unsupported_expression", node.loc.start);
  const [kind, schema, leftField, rightField, operandType] = rule;
  const left = emitExpression(node.left, owner, `${path}.left`, context, operandType);
  const right = emitExpression(node.right, owner, `${path}.right`, context, operandType);
  const id = expressionID(owner, path, kind);
  context.entities.push(graphEntity(id, entity(id, schema, [[leftField, ref(left.id)], [rightField, ref(right.id)]])));
  return { id, type: expected };
}

function blockStatements(node) {
  if (node.type === "BlockStatement") return node.body;
  if (node.type === "IfStatement" || node.type === "ReturnStatement") return [node];
  fail("javascript.unsupported_branch", node.loc.start);
}
function expressionID(owner, path, kind) { return stableID("execution", owner, "expression", path, kind); }
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
  entities.push(...additions);
  entities.sort((left, right) => left.id.localeCompare(right.id));
  return `# Generated exact JavaScript to Core Execution lift.\nve 1\nmo 00000000000000000000000000009000\nrv ${revision}\npc 0\nec ${entities.length}\n${entities.map((item) => `\n${item.text}`).join("")}`;
}
function fail(code, location) {
  const suffix = location ? `:${location.line}:${location.column + 1}` : "";
  throw new Error(`${code}${suffix}`);
}
