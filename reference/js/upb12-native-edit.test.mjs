import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {editUPB12Native} from "./upb12-native-edit.mjs";

const target="807557c5737cee0105a083f3608c7d07",expected="InitializePolicy",replacement="BuildPolicy";
for(const [language,relative,source,want] of [
  ["go","policy/seme_projected.go",`package policy\n//seme:id ${target}\nfunc InitializePolicy() {}\n`,"func BuildPolicy("],
  ["javascript","policy.js","export function InitializePolicy() {}\n","export function BuildPolicy("],
  ["lua","policy.lua",`---@seme-id ${target}\nlocal function InitializePolicy() end\nreturn { InitializePolicy = InitializePolicy }\n`,"BuildPolicy = BuildPolicy"],
])test(`applies the same identity-bound edit to ${language}`,()=>{const parent=fs.mkdtempSync(path.join(os.tmpdir(),"seme-upb12-edit-test-")),root=path.join(parent,"source"),graph=path.join(parent,"graph.json"),out=path.join(parent,"result");try{fs.mkdirSync(path.dirname(path.join(root,relative)),{recursive:true});fs.writeFileSync(path.join(root,relative),source);fs.writeFileSync(graph,JSON.stringify({Packages:[{Identity:"seme.upb12/service/policy",Sources:[{Path:"policy."+(language==="lua"?"lua":"js")}],Members:[{Identity:target,Name:expected,ExportName:expected,Callable:true}]}]}));const report=editUPB12Native({projectRoot:root,graphPath:graph,language,target,expected,replacement,destination:out});assert.equal(report.target,target);assert.match(fs.readFileSync(path.join(out,relative),"utf8"),new RegExp(want.replace(/[()]/g,"\\$&")));assert.equal(JSON.parse(fs.readFileSync(path.join(out,"SEME-EDIT.json"))).replacement,replacement)}finally{fs.rmSync(parent,{recursive:true,force:true})}});

test("rejects a stale target without partial publication",()=>{const parent=fs.mkdtempSync(path.join(os.tmpdir(),"seme-upb12-edit-stale-")),root=path.join(parent,"source"),graph=path.join(parent,"graph.json"),out=path.join(parent,"result");try{fs.mkdirSync(root);fs.writeFileSync(path.join(root,"policy.js"),"export function Different() {}\n");fs.writeFileSync(graph,JSON.stringify({Packages:[{Identity:"seme.upb12/service/policy",Sources:[{Path:"policy.js"}],Members:[{Identity:target,Name:expected,ExportName:expected,Callable:true}]}]}));assert.throws(()=>editUPB12Native({projectRoot:root,graphPath:graph,language:"javascript",target,expected,replacement,destination:out}),/occurrence/);assert.equal(fs.existsSync(out),false)}finally{fs.rmSync(parent,{recursive:true,force:true})}});
