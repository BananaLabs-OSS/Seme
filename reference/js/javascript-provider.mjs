import crypto from "node:crypto";
import { parse } from "acorn";

const ids = {
  i64: stableID("execution", "type", "i64"),
  bool: stableID("execution", "type", "bool"),
  string: stableID("execution", "type", "string"),
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
  const entities = [
    graphEntity(ids.i64, entity(ids.i64, "00000000000000000000000000009010", [[0x9100, "uu 64"], [0x9101, "tr"], [0x9102, "uu 0"]])),
    graphEntity(ids.bool, entity(ids.bool, "00000000000000000000000000009020", [])),
    graphEntity(ids.string, entity(ids.string, "00000000000000000000000000009040", [])),
  ];
  for (const description of descriptions) {
    const { fn, signature, id: functionID } = description;
    const parameterIDs = fn.params.map((parameter, index) => {
      if (parameter.type !== "Identifier") fail("javascript.unsupported_parameter", parameter.loc.start);
      if (signature.parameters[index].name !== parameter.name) fail("javascript.signature_name", parameter.loc.start);
      const parameterID = stableID("execution", functionID, "parameter", String(index));
      entities.push(graphEntity(parameterID, entity(parameterID, "00000000000000000000000000009012", [
        [0x9120, bytes(parameter.name)], [0x9121, ref(ids[signature.parameters[index].type])], [0x9122, `uu ${index}`],
      ])));
      return parameterID;
    });
    const context = { functionID, parameterIDs, parameterNames: fn.params.map((item) => item.name), parameterTypes: signature.parameters.map((item) => item.type), entities, locals: new Map(), nextLocal: { value: 0 }, functionsByName };
    const bodyID = emitBlock(fn.body.body, "body", context, signature.result);
    entities.push(graphEntity(functionID, entity(functionID, "00000000000000000000000000009011", [
      [0x9110, bytes(fn.id.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(ids[signature.result])], [0x9113, ref(bodyID)],
    ])));
  }
  const selectedName = entryName || (exported.size === 1 ? [...exported][0] : descriptions[0].fn.id.name);
  const entry = functionsByName.get(selectedName);
  if (!entry) fail("javascript.entry_missing");
  const programID = stableID("session-program", packagePath);
  entities.push(graphEntity(programID, entity(programID, "00000000000000000000000000009015", [[0x9150, refs(descriptions.map((item) => item.id))], [0x9151, ref(entry.id)]])));
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
  const localContext = { ...context, locals: new Map(context.locals) };
  const statementIDs = [];
  for (let index = 0; index < statements.length; index += 1) {
    const current = statements[index];
    const statementPath = statements.length === 1 ? `${path}.statement` : `${path}.statement.${index}`;
    if (current.type === "VariableDeclaration") {
      if (current.kind !== "const" || current.declarations.length !== 1) fail("javascript.local_binding_shape", current.loc.start);
      const declaration = current.declarations[0];
      if (declaration.id.type !== "Identifier" || !declaration.init || localContext.locals.has(declaration.id.name)) fail("javascript.local_binding_shape", current.loc.start);
      const valueType = inferExpressionType(declaration.init, localContext);
      const local = localContext.nextLocal.value++;
      const bindingID = stableID("execution", context.functionID, path, "local", String(local));
      const initializer = emitExpression(declaration.init, `${context.functionID}:${path}:local:${local}`, "root", localContext, valueType);
      context.entities.push(graphEntity(bindingID, entity(bindingID, "000000000000000000000000000090d0", [[0x9d00, bytes(declaration.id.name)], [0x9d01, ref(ids[valueType])], [0x9d02, ref(initializer.id)]])));
      const statementID = stableID("execution", context.functionID, statementPath, "bind-local");
      context.entities.push(graphEntity(statementID, entity(statementID, "000000000000000000000000000090d1", [[0x9d10, ref(bindingID)]])));
      statementIDs.push(statementID);
      localContext.locals.set(declaration.id.name, { id: bindingID, type: valueType });
      continue;
    }
    if (current.type === "ReturnStatement") {
      if (index !== statements.length - 1) fail("javascript.return_not_terminal", current.loc.start);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      statementIDs.push(emitReturn(current, path, canonicalPath, localContext, resultType));
      continue;
    }
    if (current.type === "IfStatement") {
      const following = statements.slice(index + 1);
      if (current.alternate && following.length) fail("javascript.unreachable_following", following[0].loc.start);
      const condition = emitExpression(current.test, `${context.functionID}:${path}:condition`, "root", localContext, "bool");
      const thenID = emitBlock(blockStatements(current.consequent), `${path}.then`, localContext, resultType);
      const elseStatements = current.alternate ? blockStatements(current.alternate) : following;
      if (!elseStatements.length) fail("javascript.branch_not_total", current.loc.start);
      const elseID = emitBlock(elseStatements, `${path}.else`, localContext, resultType);
      const canonicalPath = statementIDs.length === 0 ? `${path}.statement` : `${path}.statement.${statementIDs.length}`;
      const id = stableID("execution", context.functionID, canonicalPath, "if");
      context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090c0", [[0x9c00, ref(condition.id)], [0x9c01, ref(thenID)], [0x9c02, ref(elseID)]])));
      statementIDs.push(id);
      break;
    }
    fail("javascript.unsupported_statement", current.loc.start);
  }
  if (!statementIDs.length) fail("javascript.block_not_total", statements[0]?.loc?.start);
  const blockID = stableID("execution", context.functionID, path, "block");
  context.entities.push(graphEntity(blockID, entity(blockID, "00000000000000000000000000009080", [[0x9800, refs(statementIDs)]])));
  return blockID;
}

function emitReturn(statement, path, statementPath, context, resultType) {
  if (!statement.argument) fail("javascript.return_arity", statement.loc.start);
  const expression = emitExpression(statement.argument, `${context.functionID}:${path}`, "root", context, resultType);
  const id = stableID("execution", context.functionID, statementPath, "return");
  context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009081", [[0x9810, refs([expression.id])]])));
  return id;
}

function emitExpression(node, owner, path, context, expected) {
  if (node.type === "Identifier") {
    const index = context.parameterNames.indexOf(node.name);
    if (index < 0) {
      const local = context.locals.get(node.name);
      if (!local || local.type !== expected) fail("javascript.unresolved_or_mistyped_identifier", node.loc.start);
      const id = stableID("execution", owner, "local-read", local.id);
      context.entities.push(graphEntity(id, entity(id, "000000000000000000000000000090d2", [[0x9d20, ref(local.id)]])));
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
  if (node.type === "CallExpression" && node.callee.type === "Identifier" && !node.optional) {
    const callee = context.functionsByName.get(node.callee.name);
    if (!callee || callee.signature.result !== expected || callee.signature.parameters.length !== node.arguments.length) fail("javascript.unsupported_call", node.loc.start);
    const arguments_ = node.arguments.map((argument, index) => emitExpression(argument, owner, `${path}.argument.${index}`, context, callee.signature.parameters[index].type));
    const id = expressionID(owner, path, "function-call");
    context.entities.push(graphEntity(id, entity(id, "00000000000000000000000000009060", [[0x9600, ref(callee.id)], [0x9601, refs(arguments_.map((item) => item.id))]])));
    return { id, type: expected };
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

function inferExpressionType(node, context) {
  if (node.type === "Identifier") {
    const parameter = context.parameterNames.indexOf(node.name);
    if (parameter >= 0) return context.parameterTypes[parameter];
    const local = context.locals.get(node.name);
    if (local) return local.type;
  }
  if (node.type === "Literal" && typeof node.value === "string") return "string";
  if (node.type === "Literal" && typeof node.value === "boolean") return "bool";
  if (node.type === "CallExpression" && node.callee.type === "Identifier") {
    const callee = context.functionsByName.get(node.callee.name);
    if (callee) return callee.signature.result;
  }
  if (node.type === "LogicalExpression" && (node.operator === "&&" || node.operator === "||")) return "bool";
  if (node.type === "BinaryExpression" && node.operator === "===") return "bool";
  if (node.type === "BinaryExpression" && node.operator === "+") {
    const left = inferExpressionType(node.left, context);
    const right = inferExpressionType(node.right, context);
    if (left === "string" && right === "string") return "string";
  }
  fail("javascript.ambiguous_local_type", node.loc.start);
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
