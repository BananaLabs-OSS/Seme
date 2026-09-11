import crypto from "node:crypto";
import path from "node:path/posix";
import { parse } from "acorn";
import { javascriptDeclarationIdentity, javascriptRecordIdentity } from "./javascript-provider.mjs";

export function buildJavaScriptProjectGraph({ files, snapshot, projectPath, rootModule, canonicalG1 }) {
  if (!Array.isArray(files) || !snapshot || snapshot.RootIdentity !== projectPath) fail("identity");
  const units = new Map(snapshot.Units.map((unit) => [unit.Path, unit]));
  const modules = new Map();
  for (const file of files) {
    const unit = units.get(file.path);
    if (!unit || unit.Class !== "tracked" || unit.Preservation !== "semantic-projection") fail(`source:${file.path}`);
    if (Buffer.byteLength(file.source) !== unit.Size || sha256(file.source) !== unit.SHA256) fail(`digest:${file.path}`);
    let ast; const comments=[];
    try { ast = parse(file.source, { ecmaVersion: 2024, sourceType: "module", locations: true, onComment: comments }); }
    catch (error) { throw new Error(`javascript_project_graph.parse:${file.path}:${error.loc?.line ?? 0}:${(error.loc?.column ?? -1) + 1}`); }
    const identity = moduleIdentity(projectPath, file.path);
    const sourceIdentity = stableSourceIdentity(projectPath, file.path);
    const exports = new Set();
    const functions = [];
    for (const item of ast.body) {
      const declaration = item.type === "ExportNamedDeclaration" ? item.declaration : item;
      if (item.type === "ExportNamedDeclaration" && !declaration) fail(`reexport:${file.path}`);
      if (declaration?.type === "FunctionDeclaration") {
        if (item.type === "ExportNamedDeclaration") exports.add(declaration.id.name);
        functions.push(declaration);
      }
    }
    const records=comments.flatMap((comment)=>recordDeclaration(comment,projectPath));
    modules.set(file.path, { ...file, unit, ast, identity, sourceIdentity, exports, functions, records });
  }
  if (!modules.has(rootModule) || modules.size === 0 || modules.size !== files.length) fail("root");
  const signatures = canonicalSignatures(canonicalG1);
  const edges = new Map([...modules.keys()].map((key) => [key, []]));
  const packages = [];
  for (const module of modules.values()) {
    const origin = (node) => ({ SourceIdentity: module.sourceIdentity, Path: module.path, ContentDigest: module.unit.SHA256, ByteStart: node.start, ByteEnd: node.end, StartLine: node.loc.start.line, StartColumn: node.loc.start.column + 1, EndLine: node.loc.end.line, EndColumn: node.loc.end.column + 1 });
    const members = module.functions.map((fn) => {
      const identity = javascriptDeclarationIdentity(projectPath, fn.id.name);
      const signature = signatures.get(identity);
      if (!signature) fail(`signature:${module.path}:${fn.id.name}`);
      const exported = module.exports.has(fn.id.name);
      return { Identity: identity, Name: fn.id.name, ExportName: exported ? fn.id.name : "", Visibility: exported ? 2 : 0, Origin: origin(fn), Callable: true, Parameters: signature.parameters, Results: [signature.result] };
    });
    for(const record of module.records){
      if(!canonicalRecord(canonicalG1,record.Identity))fail(`record:${module.path}:${record.Name}`);
      members.push({Identity:record.Identity,Name:record.Name,ExportName:record.Name,Visibility:2,Origin:origin(record.Node),Callable:false,Parameters:[],Results:[]});
    }
    members.sort((a, b) => a.Identity.localeCompare(b.Identity));
    const imports = [];
    for (const declaration of module.ast.body.filter((item) => item.type === "ImportDeclaration")) {
      if (typeof declaration.source.value !== "string" || !declaration.source.value.startsWith(".") || declaration.specifiers.some((item) => item.type !== "ImportSpecifier")) fail(`import_shape:${module.path}`);
      const targetPath = resolveImport(module.path, declaration.source.value);
      const target = modules.get(targetPath);
      if (!target) fail(`import_missing:${module.path}:${declaration.source.value}`);
      if (targetPath === module.path) fail(`self_import:${module.path}`);
      edges.get(module.path).push(targetPath);
      for (const specifier of declaration.specifiers) {
        const imported = specifier.imported.name;
        if (!target.exports.has(imported)) fail(`import_private:${module.path}:${imported}`);
        imports.push({ Alias: specifier.local.name, Requested: declaration.source.value, Resolved: target.identity, Class: 0, Origin: origin(specifier) });
      }
    }
    imports.sort((a, b) => a.Origin.SourceIdentity.localeCompare(b.Origin.SourceIdentity) || a.Origin.ByteStart - b.Origin.ByteStart || a.Requested.localeCompare(b.Requested) || a.Alias.localeCompare(b.Alias));
    packages.push({ Identity: module.identity, Root: module.path === rootModule, Sources: [{ Identity: module.sourceIdentity, Path: module.path, ContentDigest: module.unit.SHA256, ByteSize: module.unit.Size }], Members: members, Imports: imports });
  }
  rejectCycles(edges);
  packages.sort((a, b) => a.Identity.localeCompare(b.Identity));
  return { Packages: packages };
}

function recordDeclaration(comment,projectPath){
  const match=comment.value.match(/@typedef\s+\{Object\}\s+([A-Za-z_$][\w$]*)/);
  if(!match)return [];
  return [{Name:match[1],Identity:javascriptRecordIdentity(projectPath,match[1]),Node:comment}];
}
function canonicalRecord(g1,identity){return new RegExp(`^en ${identity} 00000000000000000000000000009030 1 \\d+$`,`m`).test(g1)}

function canonicalSignatures(g1) {
  if (typeof g1 !== "string") fail("canonical");
  const entities = new Map(); let current;
  const lines = g1.split(/\r?\n/);
  for (let index = 0; index < lines.length; index++) {
    const parts = lines[index].trim().split(/\s+/);
    if (parts[0] === "en") { current = { schema: parts[2], fields: new Map() }; entities.set(parts[1], current); continue; }
    if (parts[0] !== "fi" || !current) continue;
    if (parts[2] === "rf") current.fields.set(parts[1], parts[3]);
    if (parts[2] === "li") { const values=[]; for(let n=0;n<Number(parts[3]);n++){const item=lines[++index].trim().split(/\s+/);if(item[0]!=="rf")fail("canonical_list");values.push(item[1]);} current.fields.set(parts[1], values); }
  }
  const result = new Map();
  for (const [identity, entity] of entities) if (entity.schema === "00000000000000000000000000009011") {
    const parameters = (entity.fields.get("00000000000000000000000000009111") ?? []).map((id) => entities.get(id)?.fields.get("00000000000000000000000000009121"));
    const returnType = entity.fields.get("00000000000000000000000000009112");
    if (parameters.some((id) => !id) || !returnType) fail(`canonical_signature:${identity}`);
    result.set(identity, { parameters, result: returnType });
  }
  return result;
}
function moduleIdentity(projectPath, file) { return `${projectPath}/${file.replace(/\.js$/, "")}`; }
function resolveImport(from, specifier) { const resolved=path.normalize(path.join(path.dirname(from),specifier)); if(resolved===".."||resolved.startsWith("../")||path.isAbsolute(resolved))fail(`import_escape:${from}:${specifier}`);return path.extname(resolved) ? resolved : `${resolved}.js`; }
function sha256(value){return crypto.createHash("sha256").update(value).digest("hex")}
function stableSourceIdentity(root,file){const hash=crypto.createHash("sha256").update("seme.source-inventory.identity.v1\0");for(const value of ["unit",root,file]){const size=Buffer.alloc(8);size.writeBigUInt64BE(BigInt(Buffer.byteLength(value)));hash.update(size).update(value);}return `80${hash.digest("hex").slice(0,30)}`}
function rejectCycles(edges){const state=new Map();const visit=(node)=>{if(state.get(node)===1)fail(`cycle:${node}`);if(state.get(node)===2)return;state.set(node,1);for(const next of edges.get(node)??[])visit(next);state.set(node,2)};for(const node of edges.keys())visit(node)}
function fail(code){throw new Error(`javascript_project_graph.${code}`)}
