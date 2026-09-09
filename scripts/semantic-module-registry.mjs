import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";

const idPattern = /^[0-9a-f]{32}$/;
const moduleSchema = "00000000000000000000000000000012";
const importSchema = "00000000000000000000000000000013";
const importsField = "00000000000000000000000000000121";
const exportsField = "00000000000000000000000000000122";
const importModuleField = "00000000000000000000000000000130";
const importRevisionField = "00000000000000000000000000000131";

const args = process.argv.slice(2);
let root = "modules", output = "";
for (let index = 0; index < args.length; index += 1) {
  if (args[index] === "--root") root = args[++index] ?? "";
  else if (args[index] === "--out") output = args[++index] ?? "";
  else fail("registry.unknown_argument", args[index]);
}
if (!root) fail("registry.root_missing");

const files = [];
for (const family of sortedDirectories(root)) {
  for (const revision of sortedDirectories(path.join(root, family))) {
    const candidate = path.join(root, family, revision, "module.g1");
    if (fs.existsSync(candidate)) files.push({ family, revisionDirectory: revision, path: candidate });
  }
}
if (files.length === 0) fail("registry.no_modules", root);

const modules = files.map(parseModule);
const globalOwners = new Map(), artifactPairs = new Set(), revisionArtifacts = new Map();
for (const item of modules) {
  const pair = `${item.module}:${item.revision}`;
  if (artifactPairs.has(pair)) fail("registry.duplicate_module_revision_artifact", item.module, item.revision);
  artifactPairs.add(pair);
  if (revisionArtifacts.has(item.revision)) fail("registry.duplicate_revision_artifact", item.revision, revisionArtifacts.get(item.revision), item.relativePath);
  revisionArtifacts.set(item.revision, item.relativePath);
  own(globalOwners, item.module, item.family, "module", "registry.cross_family_owned_identity_collision");
  own(globalOwners, item.revision, item.family, "revision", "registry.cross_family_owned_identity_collision");
  for (const exported of item.exports) own(globalOwners, exported, item.family, "export", "registry.cross_family_owned_identity_collision");
}
const revisionIndex = new Map(modules.map((item) => [item.revision, item]));
const moduleIndex = new Map();
for (const item of modules) {
  const entries = moduleIndex.get(item.module) ?? [];
  entries.push(item);
  moduleIndex.set(item.module, entries);
}
for (const item of modules) {
  if (new Set(item.parents).size !== item.parents.length) fail("registry.duplicate_parent_pin", item.relativePath);
  if (JSON.stringify(item.parents) !== JSON.stringify([...item.parents].sort())) fail("registry.unsorted_parent_pins", item.relativePath);
  for (const pin of item.parents) {
    const target = revisionIndex.get(pin);
    if (!target) fail("registry.parent_pin_missing", item.relativePath, pin);
    if (target.module !== item.module || target.family !== item.family) fail("registry.parent_pin_cross_lineage", item.relativePath, pin);
    if (pin === item.revision) fail("registry.parent_pin_self", item.relativePath, pin);
  }
  for (const imported of item.imports) {
    const revisions = moduleIndex.get(imported.module) ?? [];
    const target = revisions.find((entry) => entry.revision === imported.revision);
    if (!target) fail("registry.import_pin_missing", item.relativePath, imported.module, imported.revision);
  }
}
for (const item of modules) visitParents(item, new Set(), []);

const families = [];
for (const family of [...new Set(modules.map((item) => item.family))].sort()) {
  const entries = modules.filter((item) => item.family === family).sort((a, b) => a.relativePath.localeCompare(b.relativePath));
  const moduleIDs = [...new Set(entries.map((item) => item.module))];
  if (moduleIDs.length !== 1) fail("registry.family_has_multiple_modules", family, ...moduleIDs);
  families.push({
    family,
    module: moduleIDs[0],
    revisions: entries.map((item) => ({
      directory: item.revisionDirectory,
      path: item.relativePath,
      revision: item.revision,
      parents: item.parents,
      exports: item.exports,
      imports: item.imports,
      entity_count: item.entities.length,
      sha256: item.sha256,
    })),
  });
}
const report = {
  contract: "seme.semantic-module-registry/v1",
  ownership_scope: ["module", "revision", "module_export"],
  deferred: ["non_exported_entity_ownership", "cross_revision_export_schema_drift"],
  family_count: families.length,
  revision_count: modules.length,
  families,
};
const encoded = `${JSON.stringify(report, null, 2)}\n`;
if (output) fs.writeFileSync(output, encoded);
else process.stdout.write(encoded);

function parseModule(input) {
  const bytes = fs.readFileSync(input.path);
  const text = bytes.toString("utf8");
  if (Buffer.from(text, "utf8").compare(bytes) !== 0) fail("registry.invalid_utf8", input.path);
  const lines = text.split(/\r?\n/);
  const significant = lines.map((value, index) => ({ value: value.trim(), line: index + 1 })).filter((item) => item.value && !item.value.startsWith("#"));
  let cursor = 0;
  const version = header("ve"), module = header("mo"), revision = header("rv"), parentCount = countHeader("pc");
  if (version !== "1") fail("registry.unsupported_g1_version", input.path, version);
  requireID(module, "registry.malformed_module_identity", input.path);
  requireID(revision, "registry.malformed_revision_identity", input.path);
  const parents = [];
  for (let index = 0; index < parentCount; index += 1) {
    const item = significant[cursor++];
    if (!item || !idPattern.test(item.value)) fail("registry.malformed_parent_pin", input.path, String(item?.line ?? 0));
    parents.push(item.value);
  }
  const entityCount = countHeader("ec");
  const entities = [];
  while (cursor < significant.length) {
    const start = significant[cursor++], parts = start.value.split(/\s+/);
    if (parts[0] !== "en" || parts.length !== 5) fail("registry.malformed_entity", input.path, String(start.line));
    requireID(parts[1], "registry.malformed_entity_identity", input.path, String(start.line));
    requireID(parts[2], "registry.malformed_schema_identity", input.path, String(start.line));
    const fieldCount = parseCount(parts[4], "registry.malformed_field_count", input.path, String(start.line));
    const block = [];
    while (cursor < significant.length && !significant[cursor].value.startsWith("en ")) block.push(significant[cursor++]);
    const fields = parseFields(block, fieldCount, input.path);
    entities.push({ id: parts[1], schema: parts[2], fields });
  }
  if (entities.length !== entityCount) fail("registry.entity_count_mismatch", input.path, String(entityCount), String(entities.length));
  if (new Set(entities.map((entity) => entity.id)).size !== entities.length) fail("registry.duplicate_entity_in_revision", input.path);
  const declaration = entities.find((entity) => entity.id === module);
  if (!declaration || declaration.schema !== moduleSchema) fail("registry.module_declaration_missing", input.path, module);
  const exports = listField(declaration, exportsField, true, input.path, "registry.exports_missing");
  if (exports.length === 0) fail("registry.exports_empty", input.path);
  const entityIDs = new Set(entities.map((entity) => entity.id));
  for (const exported of exports) if (!entityIDs.has(exported)) fail("registry.export_entity_missing", input.path, exported);
  if (new Set(exports).size !== exports.length) fail("registry.duplicate_export", input.path);
  const importRefs = listField(declaration, importsField, false, input.path);
  if (new Set(importRefs).size !== importRefs.length) fail("registry.duplicate_import", input.path);
  const declaredImportEntities = entities.filter((entity) => entity.schema === importSchema).map((entity) => entity.id).sort();
  const referencedImportEntities = [...importRefs].sort();
  if (JSON.stringify(declaredImportEntities) !== JSON.stringify(referencedImportEntities)) fail("registry.import_ownership_mismatch", input.path);
  const imports = importRefs.map((id) => {
    const entity = entities.find((candidate) => candidate.id === id);
    if (!entity || entity.schema !== importSchema) fail("registry.import_entity_missing", input.path, id);
    const importedModule = scalarField(entity, importModuleField, "rf", input.path, "registry.import_module_missing");
    const importedRevision = scalarField(entity, importRevisionField, "by", input.path, "registry.import_revision_missing");
    requireID(importedModule, "registry.malformed_import_module", input.path, id);
    requireID(importedRevision, "registry.malformed_import_revision", input.path, id);
    return { entity: id, module: importedModule, revision: importedRevision };
  });
  return { ...input, relativePath: path.relative(root, input.path).split(path.sep).join("/"), module, revision, parents, exports, imports, entities, sha256: crypto.createHash("sha256").update(bytes).digest("hex") };

  function header(name) {
    const item = significant[cursor++], parts = item?.value.split(/\s+/) ?? [];
    if (parts.length !== 2 || parts[0] !== name) fail("registry.header_missing", input.path, name, String(item?.line ?? 0));
    return parts[1];
  }
  function countHeader(name) { return parseCount(header(name), `registry.malformed_${name}_count`, input.path); }
}

function parseFields(lines, expected, file) {
  const fields = new Map();
  for (let cursor = 0; cursor < lines.length;) {
    const item = lines[cursor++], parts = item.value.split(/\s+/);
    if (parts[0] !== "fi" || parts.length !== 4) fail("registry.malformed_field", file, String(item.line));
    requireID(parts[1], "registry.malformed_field_identity", file, String(item.line));
    if (fields.has(parts[1])) fail("registry.duplicate_field", file, parts[1]);
    if (parts[2] === "li") {
      const count = parseCount(parts[3], "registry.malformed_list_count", file, String(item.line)), refs = [];
      for (let index = 0; index < count; index += 1) {
        const child = lines[cursor++], childParts = child?.value.split(/\s+/) ?? [];
        if (childParts.length !== 2 || childParts[0] !== "rf") fail("registry.malformed_reference_list", file, String(child?.line ?? 0));
        requireID(childParts[1], "registry.malformed_reference_identity", file, String(child.line));
        refs.push(childParts[1]);
      }
      fields.set(parts[1], { kind: "li", value: refs });
    } else {
      if (!/^(?:by|rf|uu|si|tr|fa|st|rc)$/.test(parts[2])) fail("registry.malformed_field_value", file, String(item.line));
      fields.set(parts[1], { kind: parts[2], value: parts[3] });
    }
  }
  // Entity headers count schema-owned fields. Refinement evidence may add
  // extension fields to the block, so only a short block is malformed here;
  // the canonical validator owns exact schema/refinement validation.
  if (fields.size < expected) fail("registry.field_count_mismatch", file, String(expected), String(fields.size));
  return fields;
}
function listField(entity, field, required, file, code = "registry.list_missing") {
  const value = entity.fields.get(field);
  if (!value) { if (required) fail(code, file, field); return []; }
  if (value.kind !== "li") fail("registry.list_field_kind", file, field);
  return value.value;
}
function scalarField(entity, field, kind, file, code) {
  const value = entity.fields.get(field);
  if (!value) fail(code, file, entity.id);
  if (value.kind !== kind) fail("registry.scalar_field_kind", file, entity.id, field);
  return value.value;
}
function sortedDirectories(directory) {
  if (!fs.existsSync(directory)) return [];
  return fs.readdirSync(directory, { withFileTypes: true }).filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort();
}
function visitParents(item, active, chain) {
  if (active.has(item.revision)) fail("registry.parent_cycle", ...chain, item.revision);
  const next = new Set(active); next.add(item.revision);
  for (const parent of item.parents) visitParents(revisionIndex.get(parent), next, [...chain, item.revision]);
}
function parseCount(value, code, ...context) {
  if (!/^(?:0|[1-9][0-9]*)$/.test(value)) fail(code, ...context, value);
  return Number(value);
}
function requireID(value, code, ...context) { if (!idPattern.test(value)) fail(code, ...context, value); }
function own(index, id, family, kind, code) {
  const key = `${kind}:${id}`;
  const prior = index.get(key);
  if (prior && prior.family !== family) fail(code, id, prior.family, prior.kind, family, kind);
  index.set(key, prior ?? { family, kind });
}
function fail(code, ...details) { throw new Error([code, ...details].join(":")); }
