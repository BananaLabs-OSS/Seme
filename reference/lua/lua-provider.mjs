import crypto from "node:crypto";

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

/**
 * Lift the deliberately bounded Lua 5.1 / LuaJIT profile directly to Seme.
 * `sources` is an ordered array of { name, source }; file names never affect
 * semantic identity.
 */
export function liftLua({ sources, packagePath, revision, moduleG1, entryName }) {
  if (!Array.isArray(sources) || sources.length === 0) fail("lua.requires_sources");
  if (!packagePath || !Number.isSafeInteger(revision) || revision < 1) fail("lua.invalid_snapshot");
  const declarations = sources.flatMap(({ name, source }) => parseSource(source, name));
  const byName = new Map();
  for (const declaration of declarations) {
    if (byName.has(declaration.name)) fail("lua.duplicate_function", declaration.location);
    byName.set(declaration.name, declaration);
  }
  const entry = entryName ? byName.get(entryName) : declarations.find((item) => item.exported);
  if (!entry) fail("lua.entry_not_found");
  if (!entry.exported) fail("lua.entry_must_be_global", entry.location);

  const additions = [graphEntity(ids.bool, entity(ids.bool, schema.boolType, []))];
  const descriptions = declarations.map((declaration) => ({
    ...declaration,
    id: stableID("session-declaration", packagePath, declaration.name),
  }));
  const descriptionsByName = new Map(descriptions.map((item) => [item.name, item]));

  for (const description of descriptions) {
    const parameterIDs = description.parameters.map((parameter, index) => {
      const id = stableID("execution", description.id, "parameter", String(index));
      additions.push(graphEntity(id, entity(id, schema.parameter, [
        [0x9120, bytes(parameter)], [0x9121, ref(ids.bool)], [0x9122, `uu ${index}`],
      ])));
      return id;
    });
    const context = { description, parameterIDs, descriptionsByName, additions };
    const expression = emitExpression(description.expression, context, "body.statement.expression");
    const returnedID = stableID("execution", description.id, "body.statement", "return");
    const blockID = stableID("execution", description.id, "body", "block");
    additions.push(graphEntity(returnedID, entity(returnedID, schema.returned, [[0x9810, refs([expression])]])));
    additions.push(graphEntity(blockID, entity(blockID, schema.block, [[0x9800, refs([returnedID])]])));
    additions.push(graphEntity(description.id, entity(description.id, schema.function, [
      [0x9110, bytes(description.name)], [0x9111, refs(parameterIDs)], [0x9112, ref(ids.bool)], [0x9113, ref(blockID)],
    ])));
  }
  const functionIDs = descriptions.map((item) => item.id).sort();
  const programID = stableID("session-program", packagePath);
  additions.push(graphEntity(programID, entity(programID, schema.program, [
    [0x9150, refs(functionIDs)], [0x9151, ref(stableID("session-declaration", packagePath, entry.name))],
  ])));
  return compose(moduleG1, stableID("session-revision", packagePath, String(revision)), additions);
}

function parseSource(source, file) {
  if (typeof source !== "string" || typeof file !== "string" || !file) fail("lua.invalid_source");
  const lines = source.replace(/\r\n?/g, "\n").split("\n");
  const declarations = [];
  let annotations = [];
  for (let index = 0; index < lines.length;) {
    const trimmed = lines[index].trim();
    if (!trimmed || trimmed.startsWith("--") && !trimmed.startsWith("---@")) { index += 1; continue; }
    if (trimmed.startsWith("---@")) { annotations.push({ text: trimmed, line: index + 1 }); index += 1; continue; }
    const match = /^(local\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(([^)]*)\)\s*$/.exec(trimmed);
    if (!match) fail("lua.unsupported_top_level", { file, line: index + 1, column: 1 });
    const location = { file, line: index + 1, column: lines[index].indexOf("function") + 1 };
    const parameters = match[3].trim() ? match[3].split(",").map((item) => item.trim()) : [];
    if (parameters.some((item) => !/^[A-Za-z_][A-Za-z0-9_]*$/.test(item))) fail("lua.unsupported_parameter", location);
    validateAnnotations(annotations, parameters, location);
    annotations = [];
    index += 1;
    const body = [];
    while (index < lines.length && lines[index].trim() !== "end") { body.push({ text: lines[index], line: index + 1 }); index += 1; }
    if (index >= lines.length) fail("lua.unclosed_function", location);
    if (body.filter((item) => item.text.trim()).length !== 1) fail("lua.function_body_profile", location);
    const statement = body.find((item) => item.text.trim());
    const returned = /^\s*return\s+(.+?)\s*$/.exec(statement.text);
    if (!returned) fail("lua.requires_return", { file, line: statement.line, column: 1 });
    declarations.push({ name: match[2], parameters, expression: parseExpression(returned[1], file, statement.line), exported: !match[1], location });
    index += 1;
  }
  if (annotations.length) fail("lua.orphan_annotation", { file, line: annotations[0].line, column: 1 });
  return declarations;
}

function validateAnnotations(annotations, parameters, location) {
  const declared = annotations.filter((item) => item.text.startsWith("---@param ")).map((item) => /^---@param\s+([A-Za-z_][A-Za-z0-9_]*)\s+boolean$/.exec(item.text));
  const result = annotations.filter((item) => item.text.startsWith("---@return "));
  if (declared.some((item) => !item) || result.length !== 1 || result[0].text !== "---@return boolean") fail("lua.unsupported_annotation", location);
  if (declared.length !== parameters.length || declared.some((item, index) => item[1] !== parameters[index])) fail("lua.signature_mismatch", location);
  if (annotations.length !== declared.length + 1) fail("lua.unsupported_annotation", location);
}

function parseExpression(text, file, line) {
  const identifier = /^([A-Za-z_][A-Za-z0-9_]*)$/.exec(text);
  if (identifier) return { kind: "identifier", name: identifier[1], location: { file, line, column: 1 } };
  const call = /^([A-Za-z_][A-Za-z0-9_]*)\s*\((.*)\)$/.exec(text);
  if (call) {
    const arguments_ = call[2].trim() ? call[2].split(",").map((item) => parseExpression(item.trim(), file, line)) : [];
    return { kind: "call", name: call[1], arguments: arguments_, location: { file, line, column: 1 } };
  }
  fail("lua.unsupported_expression", { file, line, column: 1 });
}

function emitExpression(expression, context, path) {
  if (expression.kind === "identifier") {
    const index = context.description.parameters.indexOf(expression.name);
    if (index < 0) fail("lua.unknown_identifier", expression.location);
    const id = stableID("execution", context.description.id, "expression", path, "parameter-read");
    context.additions.push(graphEntity(id, entity(id, schema.read, [[0x9130, ref(context.parameterIDs[index])]])));
    return id;
  }
  const callee = context.descriptionsByName.get(expression.name);
  if (!callee) fail("lua.unknown_call", expression.location);
  if (callee.parameters.length !== expression.arguments.length) fail("lua.call_arity", expression.location);
  const arguments_ = expression.arguments.map((item, index) => emitExpression(item, context, `${path}.argument.${index}`));
  const id = stableID("execution", context.description.id, "expression", path, "call");
  context.additions.push(graphEntity(id, entity(id, schema.call, [[0x9600, ref(callee.id)], [0x9601, refs(arguments_)]])));
  return id;
}

const ids = { bool: stableID("execution", "type", "bool") };
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
