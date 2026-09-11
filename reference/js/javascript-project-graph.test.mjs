import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript, liftJavaScriptPackage } from "./javascript-provider.mjs";
import { buildJavaScriptProjectGraph } from "./javascript-project-graph.mjs";

const projectPath="example.test/javascript-upb02";
const files=["application.js","math/sum.js"].map((path)=>({path,source:fs.readFileSync(new URL(`../../fixtures/javascript-upb02-modules/${path}`,import.meta.url),"utf8")}));
const moduleG1=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8");
const canonicalG1=liftJavaScriptPackage({files,packagePath:projectPath,revision:1,moduleG1,entryName:"Run"});
const snapshot={RootIdentity:projectPath,Units:files.map((file)=>({Path:file.path,Class:"tracked",Preservation:"semantic-projection",Size:Buffer.byteLength(file.source),SHA256:crypto.createHash("sha256").update(file.source).digest("hex")}))};
const build=(input=files)=>buildJavaScriptProjectGraph({files:input,snapshot,projectPath,rootModule:"application.js",canonicalG1});

test("preserves module ownership, public/private visibility, imports, and signatures",()=>{
  const graph=build();assert.equal(graph.Packages.length,2);const app=graph.Packages.find((p)=>p.Root);const math=graph.Packages.find((p)=>!p.Root);assert.equal(app.Imports[0].Requested,"./math/sum.js");assert.equal(app.Imports[0].Resolved,math.Identity);assert.deepEqual(math.Members.map((m)=>[m.Name,m.Visibility]),[["Sum",2],["normalize",0]]);assert.equal(app.Members[0].Parameters.length,2);
});
test("is independent of file enumeration order",()=>assert.deepEqual(build([...files].reverse()),build()));
test("rejects missing and private imports",()=>{
  const missingSource=files[0].source.replace("./math/sum.js","./missing.js");const missing={...files[0],source:missingSource};const missingSnapshot={...snapshot,Units:[{...snapshot.Units[0],Size:Buffer.byteLength(missingSource),SHA256:crypto.createHash("sha256").update(missingSource).digest("hex")},snapshot.Units[1]]};assert.throws(()=>buildJavaScriptProjectGraph({files:[missing,files[1]],snapshot:missingSnapshot,projectPath,rootModule:"application.js",canonicalG1}),/javascript_project_graph\.import_missing/);
  const privateSource=files[0].source.replaceAll("Sum","normalize");const privateFile={...files[0],source:privateSource};const privateSnapshot={...snapshot,Units:[{...snapshot.Units[0],Size:Buffer.byteLength(privateSource),SHA256:crypto.createHash("sha256").update(privateSource).digest("hex")},snapshot.Units[1]]};assert.throws(()=>buildJavaScriptProjectGraph({files:[privateFile,files[1]],snapshot:privateSnapshot,projectPath,rootModule:"application.js",canonicalG1}),/javascript_project_graph\.import_private/);
});
test("rejects cycles and project escapes",()=>{
  const cyclicSource=`import { Run } from "../application.js";\n${files[1].source}`;const cyclic={...files[1],source:cyclicSource};const cyclicSnapshot={...snapshot,Units:[snapshot.Units[0],{...snapshot.Units[1],Size:Buffer.byteLength(cyclicSource),SHA256:crypto.createHash("sha256").update(cyclicSource).digest("hex")}]};assert.throws(()=>buildJavaScriptProjectGraph({files:[files[0],cyclic],snapshot:cyclicSnapshot,projectPath,rootModule:"application.js",canonicalG1}),/javascript_project_graph\.cycle/);
  const escapeSource=files[0].source.replace("./math/sum.js","../outside.js");const escaped={...files[0],source:escapeSource};const escapeSnapshot={...snapshot,Units:[{...snapshot.Units[0],Size:Buffer.byteLength(escapeSource),SHA256:crypto.createHash("sha256").update(escapeSource).digest("hex")},snapshot.Units[1]]};assert.throws(()=>buildJavaScriptProjectGraph({files:[escaped,files[1]],snapshot:escapeSnapshot,projectPath,rootModule:"application.js",canonicalG1}),/javascript_project_graph\.import_escape/);
});
test("publishes authenticated structural record ownership",()=>{
  const source=`/** @typedef {Object} Settings\n * @property {string} Namespace\n */\nexport class Settings { constructor(Namespace) { this.Namespace = Namespace; } }\n/** @param {Settings} value @returns {string} */\nexport function Read(value) { return value.Namespace; }`;
  const path="application.js",project="example.test/javascript-record-project";
  const input=[{path,source}],canonical=liftJavaScript({source,packagePath:project,revision:1,moduleG1,entryName:"Read"});
  const snap={RootIdentity:project,Units:[{Path:path,Class:"tracked",Preservation:"semantic-projection",Size:Buffer.byteLength(source),SHA256:crypto.createHash("sha256").update(source).digest("hex")}]};
  const graph=buildJavaScriptProjectGraph({files:input,snapshot:snap,projectPath:project,rootModule:path,canonicalG1:canonical});
  const record=graph.Packages[0].Members.find((member)=>!member.Callable);
  assert.equal(record.Name,"Settings");assert.equal(record.Visibility,2);assert.match(record.Identity,/^80[0-9a-f]{30}$/);assert.equal(record.Origin.Path,path);
});
