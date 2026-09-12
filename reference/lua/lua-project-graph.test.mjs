import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import test from "node:test";
import {liftLua} from "./lua-provider.mjs";
import {buildLuaProjectGraph} from "./lua-project-graph.mjs";

const projectPath="example.test/lua-upb02",paths=["application.lua","math/sum.lua"];
const files=paths.map(path=>({path,source:fs.readFileSync(new URL(`../../fixtures/lua-upb02-modules/${path}`,import.meta.url),"utf8")}));
const moduleG1=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8");
const canonicalG1=liftLua({sources:files.map(({path,source})=>({name:path,source})),packagePath:projectPath,revision:1,moduleG1,entryName:"Run"});
const snapshotFor=input=>({RootIdentity:projectPath,Units:input.map(file=>({Path:file.path,Class:"tracked",Preservation:"semantic-projection",Size:Buffer.byteLength(file.source),SHA256:crypto.createHash("sha256").update(file.source).digest("hex")}))});
const build=(input=files)=>buildLuaProjectGraph({files:input,snapshot:snapshotFor(input),projectPath,rootModule:"application.lua",canonicalG1});

test("preserves native require, exports, visibility, ownership, and signatures",()=>{const graph=build(),app=graph.Packages.find(p=>p.Root),math=graph.Packages.find(p=>!p.Root);assert.equal(graph.Packages.length,2);assert.equal(app.Imports[0].Requested,"math.sum");assert.equal(app.Imports[0].Resolved,math.Identity);assert.equal(app.Imports[0].Alias,"Sum");assert.deepEqual(math.Members.map(m=>[m.Name,m.Visibility]),[["normalize",0],["Sum",2]]);assert.equal(app.Members[0].Parameters.length,2);});
test("is independent of file enumeration order",()=>assert.deepEqual(build([...files].reverse()),build()));
test("rejects missing and private imports",()=>{const missing=files.map(file=>file.path==="application.lua"?{...file,source:file.source.replace("math.sum","math.missing")}:file);assert.throws(()=>build(missing),/import_missing/);const privateUse=files.map(file=>file.path==="application.lua"?{...file,source:file.source.replace("math.Sum","math.normalize")}:file);assert.throws(()=>build(privateUse),/import_private/);});
test("rejects cycles and duplicate exports",()=>{const cyclic=files.map(file=>file.path==="math/sum.lua"?{...file,source:`local app = require("application")\n${file.source}\nlocal _ = app.Run\n`}:file);assert.throws(()=>build(cyclic),/cycle/);const duplicate=files.map(file=>file.path==="math/sum.lua"?{...file,source:file.source.replace("return { Sum = Sum }","return { Sum = Sum, Sum = Sum }")}:file);assert.throws(()=>build(duplicate),/duplicate_export/);});
test("rejects digest drift",()=>assert.throws(()=>buildLuaProjectGraph({files,snapshot:snapshotFor(files.map((file,index)=>index?file:{...file,source:file.source+" "})),projectPath,rootModule:"application.lua",canonicalG1}),/digest/));
