import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import posix from "node:path/posix";
import { spawnSync } from "node:child_process";
import { parse } from "acorn";
import { javascriptDeclarationIdentity, liftJavaScriptPackage } from "./javascript-provider.mjs";

const metadataDirectory = ".seme-reconciliation-v1";
const identifier = /^[A-Za-z_$][\w$]*$/;

export function reconcileJavaScriptProject(options) {
  const { project, destination, projectPath, files, structuredReferences = [], moduleG1, entryName, target, expected, replacement, revision, baseRevision, nativeRunner, nativeArguments = [] } = options;
  if (![project, destination, projectPath, moduleG1, entryName, target, expected, replacement, baseRevision, nativeRunner].every((value) => typeof value === "string" && value) || !Number.isSafeInteger(revision) || revision < 2) fail("options");
  if (!path.isAbsolute(project) || !path.isAbsolute(destination) || !Array.isArray(files) || files.length < 2 || !identifier.test(expected) || !identifier.test(replacement) || expected === replacement) fail("options");
  if (target !== javascriptDeclarationIdentity(projectPath, expected)) fail("target_identity");
  const root = fs.realpathSync(project);
  if (root !== project || fs.existsSync(destination) || path.resolve(destination).startsWith(`${root}${path.sep}`)) fail("path");
  const normalized = [...new Set(files.map(normalizeFile))].sort();
  if (normalized.length !== files.length) fail("files");
  const sources = new Map(normalized.map((relative) => [relative, strictRead(path.join(root, relative)).toString("utf8")]));
  const structuredSources = new Map(structuredReferences.map(normalizeDataFile).sort().map((relative) => [relative, strictRead(path.join(root, relative)).toString("utf8")]));
  if (projectRevision(sources, structuredSources) !== baseRevision) fail("stale_native_revision");
  const programs = new Map([...sources].map(([relative, source]) => [relative, parseModule(source, relative)]));
  const declaration = findDeclaration(programs, expected);
  if (!declaration || javascriptDeclarationIdentity(projectPath, declaration.node.id.name) !== target) fail("declaration");
  rejectTopLevelCollision(programs.get(declaration.file), replacement);

  const replacements = new Map(normalized.map((relative) => [relative, []]));
  addReplacement(replacements, declaration.file, declaration.node.id, "declaration");
  let crossModule = 0;
  for (const [relative, program] of programs) {
    if (relative === declaration.file) {
      for (const node of identifierReferences(program, expected)) addReplacement(replacements, relative, node, "reference");
      continue;
    }
    for (const item of program.body.filter((node) => node.type === "ImportDeclaration")) {
      if (resolveImport(relative, item.source.value) !== declaration.file) continue;
      for (const specifier of item.specifiers) {
        if (specifier.type !== "ImportSpecifier" || specifier.imported.name !== expected) continue;
        if (specifier.local.name === expected && topLevelNames(program).has(replacement)) fail("collision");
        addReplacement(replacements, relative, specifier.imported, "imported");
        const local = specifier.local.name;
        if (local === expected) {
          addReplacement(replacements, relative, specifier.local, "binding");
          for (const node of identifierReferences(program, local)) addReplacement(replacements, relative, node, "reference");
        }
        crossModule += 1;
      }
    }
  }
  if (crossModule === 0) fail("cross_module_reference");
  const changed = new Map();
  const occurrences = [];
  for (const [relative, edits] of replacements) {
    const unique = uniqueEdits(edits);
    let source = sources.get(relative);
    for (const edit of [...unique].sort((a, b) => b.start - a.start)) source = source.slice(0, edit.start) + replacement + source.slice(edit.end);
    changed.set(relative, source);
    let shift=0;for (const edit of unique){const resultStart=edit.start+shift;occurrences.push({path:relative,start:edit.start,end:edit.end,result_start:resultStart,result_end:resultStart+replacement.length,role:edit.role});shift+=replacement.length-(edit.end-edit.start);}
  }
  const nativePackage=`${projectPath}/${declaration.file.replace(/\.js$/,"")}`;
  for(const relative of structuredReferences.map(normalizeDataFile)){
    const source=structuredSources.get(relative);let value;try{value=JSON.parse(source);}catch{fail("structured_json");}
    let semanticMatches=0;const visit=(node)=>{if(!node||typeof node!=="object")return;if(!Array.isArray(node)&&node.package===nativePackage&&node.name===expected)semanticMatches+=1;for(const child of Object.values(node))visit(child);};visit(value);
    if(semanticMatches===0)continue;if(semanticMatches!==1)fail("structured_reference_cardinality");
    const encoded=JSON.stringify(expected),index=source.indexOf(encoded);if(index<0||source.indexOf(encoded,index+encoded.length)>=0)fail("structured_reference_encoding");
    const start=index+1,end=start+expected.length;changed.set(relative,source.slice(0,start)+replacement+source.slice(end));occurrences.push({path:relative,start,end,result_start:start,result_end:start+replacement.length,role:"structured-reference"});
  }
  // The bounded cumulative fixture has one exported declaration and one named
  // cross-module import. A referenced local binding adds further occurrences,
  // but an unused native import is still a typed module-edge reference.
  if (occurrences.length < 2) fail("occurrences");

  const moduleSource = strictRead(moduleG1).toString("utf8");
  const prior = liftJavaScriptPackage({ files: normalized.map((relative) => ({ path: relative, source: sources.get(relative) })), packagePath: projectPath, revision: revision - 1, moduleG1: moduleSource, entryName });
  const evidence = { version: 1, packagePath: projectPath, renames: [{ previousName: expected, currentName: replacement, identity: target }] };
  const result = liftJavaScriptPackage({ files: normalized.map((relative) => ({ path: relative, source: changed.get(relative) })), packagePath: projectPath, revision, moduleG1: moduleSource, entryName, identityEvidence: evidence });
  if (canonicalName(prior, target) !== expected || canonicalName(result, target) !== replacement) fail("canonical_transition");

  const parent = fs.realpathSync(path.dirname(destination));
  if (parent !== path.dirname(destination)) fail("destination_parent");
  const stage = fs.mkdtempSync(path.join(parent, ".seme-javascript-reconcile-"));
  let published = false;
  try {
    const candidate = path.join(stage, "project");
    copyTree(root, candidate);
    for (const [relative, source] of changed) fs.writeFileSync(path.join(candidate, relative), source, { flag: "w" });
    const priorNative = runNative(nativeRunner, root, nativeArguments);
    const resultNative = runNative(nativeRunner, candidate, nativeArguments);
    if (!priorNative.equals(resultNative)) fail("native_parity");
    const transcript = Buffer.from(`seme-javascript-native-validation-v1\nrunner ${path.basename(nativeRunner)}\nsha256 ${sha256(priorNative)}\n`);
    const report = { version: 1, language: "javascript", project: projectPath, client_revision: revision, target, field: "00000000000000000000000000009110", expected, replacement, occurrences: occurrences.sort((a, b) => a.path.localeCompare(b.path) || a.start - b.start || a.end - b.end) };
    const artifacts = {
      "identity-evidence.json": encode(evidence), "native-validation.txt": transcript,
      "prior-provider.g1": Buffer.from(prior), "projection-report.json": encode(report), "result-provider.g1": Buffer.from(result),
    };
    const metadata = path.join(candidate, metadataDirectory);
    fs.mkdirSync(metadata);
    for (const [name, value] of Object.entries(artifacts)) fs.writeFileSync(path.join(metadata, name), value, { flag: "wx" });
    fs.writeFileSync(path.join(metadata, "COMPLETE.sha256"), manifest(artifacts), { flag: "wx" });
    fs.renameSync(candidate, destination);
    published = true;
    return { report, evidence, prior, result, transcript };
  } finally {
    if (!published) fs.rmSync(stage, { recursive: true, force: true });
    else fs.rmdirSync(stage);
  }
}

// javascriptProjectRevision binds a semantic edit to the exact complete native
// snapshot that the editor displayed. Language-neutral Patch authority remains
// in Seme; this source-byte precondition belongs to the JavaScript provider.
export function javascriptProjectRevision({project, files, structuredReferences=[]}) {
  const root=fs.realpathSync(project);if(root!==project)fail("revision_root");
  const normalized=[...new Set(files.map(normalizeFile))].sort();if(normalized.length!==files.length)fail("files");
  const data=[...new Set(structuredReferences.map(normalizeDataFile))].sort();if(data.length!==structuredReferences.length)fail("structured_files");
  return projectRevision(new Map(normalized.map((relative)=>[relative,strictRead(path.join(root,relative)).toString("utf8")])),new Map(data.map((relative)=>[relative,strictRead(path.join(root,relative)).toString("utf8")])))
}

// JavaScriptProjectSession processes complete editor snapshots monotonically.
// Invalid newer snapshots retain the most recent valid canonical graph; stale
// snapshots never replace either the accepted revision or last-valid state.
export class JavaScriptProjectSession {
  constructor({projectPath,moduleG1,entryName}) {if(![projectPath,moduleG1,entryName].every((v)=>typeof v==="string"&&v))fail("session_options");this.projectPath=projectPath;this.moduleG1=moduleG1;this.entryName=entryName;this.lastSeen=0;this.lastValidRevision=0;this.lastValid="";}
  apply({revision,files,identityEvidence}) {
    if(!Number.isSafeInteger(revision)||revision<1||!Array.isArray(files))fail("snapshot");
    if(revision<=this.lastSeen)return {accepted:false,valid:false,disposition:"rejected-stale",revision,lastValidRevision:this.lastValidRevision,canonicalG1:this.lastValid,diagnostics:[{code:"session.stale_revision"}]};
    this.lastSeen=revision;
    try {const canonicalG1=liftJavaScriptPackage({files,packagePath:this.projectPath,revision,moduleG1:this.moduleG1,entryName:this.entryName,identityEvidence});this.lastValid=canonicalG1;this.lastValidRevision=revision;return {accepted:true,valid:true,disposition:"accepted-valid",revision,lastValidRevision:revision,canonicalG1,diagnostics:[]};}
    catch(error){return {accepted:true,valid:false,disposition:"accepted-invalid",revision,lastValidRevision:this.lastValidRevision,canonicalG1:this.lastValid,diagnostics:[{code:"javascript.lift",message:String(error?.message??error)}]};}
  }
}

export function readJavaScriptReconciliation({ project, projectPath, files, structuredReferences = [], moduleG1, entryName, nativeRunner }) {
  if (![project,projectPath,moduleG1,entryName,nativeRunner].every((value)=>typeof value==="string"&&value)||!path.isAbsolute(project)||!Array.isArray(files))fail("read_options");
  const root=fs.realpathSync(project);if(root!==project)fail("read_root");
  const metadata=path.join(root,metadataDirectory),names=["identity-evidence.json","native-validation.txt","prior-provider.g1","projection-report.json","result-provider.g1"];
  if(!fs.statSync(metadata).isDirectory()||fs.readdirSync(metadata).sort().join("\0")!==["COMPLETE.sha256",...names].sort().join("\0"))fail("metadata_files");
  const artifacts=Object.fromEntries(names.map((name)=>[name,strictRead(path.join(metadata,name))]));
  if(!strictRead(path.join(metadata,"COMPLETE.sha256")).equals(manifest(artifacts)))fail("metadata_manifest");
  let evidence,report;try{evidence=JSON.parse(artifacts["identity-evidence.json"]);report=JSON.parse(artifacts["projection-report.json"]);}catch{fail("metadata_json");}
  if(report?.version!==1||report.language!=="javascript"||report.project!==projectPath||report.client_revision<2||report.field!=="00000000000000000000000000009110"||!Array.isArray(report.occurrences)||report.occurrences.length<2||evidence?.version!==1||evidence.packagePath!==projectPath||evidence.renames?.length!==1)fail("metadata_records");
  const rename=evidence.renames[0];if(rename.identity!==report.target||rename.previousName!==report.expected||rename.currentName!==report.replacement||canonicalName(artifacts["prior-provider.g1"].toString("utf8"),report.target)!==report.expected||canonicalName(artifacts["result-provider.g1"].toString("utf8"),report.target)!==report.replacement)fail("metadata_transition");
  const normalized=[...new Set(files.map(normalizeFile))].sort();if(normalized.length!==files.length)fail("files");
  const sources=normalized.map((relative)=>({path:relative,source:strictRead(path.join(root,relative)).toString("utf8")}));
  const reproduced=liftJavaScriptPackage({files:sources,packagePath:projectPath,revision:report.client_revision,moduleG1:strictRead(moduleG1).toString("utf8"),entryName,identityEvidence:evidence});
  if(!artifacts["result-provider.g1"].equals(Buffer.from(reproduced)))fail("metadata_reingest");
  const allSources=new Map(sources.map((item)=>[item.path,item.source]));for(const relative of structuredReferences.map(normalizeDataFile))allSources.set(relative,strictRead(path.join(root,relative)).toString("utf8"));
  for(const occurrence of report.occurrences){const source=allSources.get(occurrence.path);if(!source||![occurrence.start,occurrence.end,occurrence.result_start,occurrence.result_end].every(Number.isSafeInteger)||occurrence.start<0||occurrence.end<=occurrence.start||occurrence.result_end<=occurrence.result_start||source.slice(occurrence.result_start,occurrence.result_end)!==report.replacement)fail("metadata_occurrence");}
  const native=runNative(nativeRunner,root,[]),expectedTranscript=Buffer.from(`seme-javascript-native-validation-v1\nrunner ${path.basename(nativeRunner)}\nsha256 ${sha256(native)}\n`);if(!artifacts["native-validation.txt"].equals(expectedTranscript))fail("metadata_native");
  return {report,evidence,prior:artifacts["prior-provider.g1"].toString("utf8"),result:reproduced,transcript:artifacts["native-validation.txt"]};
}

function parseModule(source, relative) { try { return parse(source, { ecmaVersion: 2024, sourceType: "module" }); } catch (error) { throw new Error(`javascript_reconcile.parse:${relative}:${error.loc?.line ?? 0}:${(error.loc?.column ?? -1) + 1}`); } }
function findDeclaration(programs, name) { const found=[]; for (const [file, program] of programs) for (const item of program.body) if (item.type === "ExportNamedDeclaration" && item.declaration?.type === "FunctionDeclaration" && item.declaration.id.name === name) found.push({file,node:item.declaration}); if(found.length!==1)fail("declaration_cardinality");return found[0]; }
function topLevelNames(program) { const names=new Set(); for(const item of program.body){const node=item.type==="ExportNamedDeclaration"?item.declaration:item;if(node?.id?.name)names.add(node.id.name);if(node?.type==="VariableDeclaration")for(const d of node.declarations)if(d.id.type==="Identifier")names.add(d.id.name);if(node?.type==="ImportDeclaration")for(const s of node.specifiers)names.add(s.local.name);}return names; }
function rejectTopLevelCollision(program, replacement) { if(topLevelNames(program).has(replacement))fail("collision"); }
function identifierReferences(program, name) { const result=[]; walk(program,null,null,(node,parent,key)=>{if(node.type!=="Identifier"||node.name!==name)return;if(parent?.type==="ImportSpecifier"||parent?.type==="FunctionDeclaration"&&key==="id"||parent?.type==="ClassDeclaration"&&key==="id"||parent?.type==="VariableDeclarator"&&key==="id"||parent?.type==="MemberExpression"&&!parent.computed&&key==="property"||parent?.type==="Property"&&!parent.computed&&key==="key"||parent?.type==="LabeledStatement"||parent?.type==="BreakStatement"||parent?.type==="ContinueStatement")return;result.push(node);});return result; }
function walk(node,parent,key,visit){if(!node||typeof node!=="object")return;visit(node,parent,key);for(const [childKey,value] of Object.entries(node)){if(childKey==="start"||childKey==="end"||childKey==="loc")continue;if(Array.isArray(value))for(const child of value)walk(child,node,childKey,visit);else walk(value,node,childKey,visit);}}
function addReplacement(map,file,node,role){map.get(file).push({start:node.start,end:node.end,role});}
function uniqueEdits(edits){const seen=new Map();for(const edit of edits){const key=`${edit.start}:${edit.end}`;const prior=seen.get(key);if(prior&&prior.role!==edit.role)continue;seen.set(key,edit);}const result=[...seen.values()].sort((a,b)=>a.start-b.start);for(let i=1;i<result.length;i++)if(result[i].start<result[i-1].end)fail("overlap");return result;}
function normalizeFile(value){if(typeof value!=="string")fail("file");const result=posix.normalize(value);if(result!==value||result===".."||result.startsWith("../")||posix.isAbsolute(result)||!result.endsWith(".js"))fail("file");return result;}
function normalizeDataFile(value){if(typeof value!=="string")fail("structured_file");const result=posix.normalize(value);if(result!==value||result===".."||result.startsWith("../")||posix.isAbsolute(result)||!result.endsWith(".json"))fail("structured_file");return result;}
function resolveImport(from,specifier){if(typeof specifier!=="string"||!specifier.startsWith("."))return "";const value=posix.normalize(posix.join(posix.dirname(from),specifier));return posix.extname(value)?value:`${value}.js`;}
function strictRead(file){const real=fs.realpathSync(file);if(real!==file||!fs.statSync(file).isFile())fail("regular_file");return fs.readFileSync(file);}
function copyTree(source,destination){fs.mkdirSync(destination);for(const entry of fs.readdirSync(source,{withFileTypes:true}).sort((a,b)=>a.name.localeCompare(b.name))){if(entry.name===metadataDirectory)continue;const from=path.join(source,entry.name),to=path.join(destination,entry.name);if(entry.isSymbolicLink())fail("symlink");if(entry.isDirectory())copyTree(from,to);else if(entry.isFile())fs.copyFileSync(from,to,fs.constants.COPYFILE_EXCL);else fail("file_type");}}
function runNative(runner,root,args){const result=spawnSync(process.execPath,[runner,root,...args],{encoding:null,maxBuffer:64<<20,env:{...process.env}});if(result.status!==0||result.signal||result.stderr.length!==0)throw new Error(`javascript_reconcile.native_validation:${result.status}:${result.stderr.toString("utf8").trim()}`);return result.stdout;}
function canonicalName(g1,target){const lines=g1.split(/\r?\n/);let active=false;for(const line of lines){const parts=line.trim().split(/\s+/);if(parts[0]==="en")active=parts[1]===target;if(active&&parts[0]==="fi"&&parts[1]==="00000000000000000000000000009110"&&parts[2]==="by")return Buffer.from(parts[3],"hex").toString("utf8");}return "";}
function encode(value){return Buffer.from(`${JSON.stringify(value,null,2)}\n`);}
function sha256(value){return crypto.createHash("sha256").update(value).digest("hex");}
function projectRevision(sources,structured){const hash=crypto.createHash("sha256");hash.update("seme-javascript-native-revision-v1\0");for(const [name,value] of [...sources,...structured].sort((a,b)=>a[0].localeCompare(b[0]))){hash.update(String(Buffer.byteLength(name)));hash.update(":");hash.update(name);hash.update(":");hash.update(String(Buffer.byteLength(value)));hash.update(":");hash.update(value);}return hash.digest("hex");}
function manifest(artifacts){return Buffer.from(`seme-javascript-reconciliation-v1\n${Object.keys(artifacts).sort().map((name)=>`${name} ${sha256(artifacts[name])}`).join("\n")}\n`);}
function fail(code){throw new Error(`javascript_reconcile.${code}`);}
