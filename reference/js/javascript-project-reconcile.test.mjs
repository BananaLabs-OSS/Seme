import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { javascriptDeclarationIdentity } from "./javascript-provider.mjs";
import { reconcileJavaScriptProject } from "./javascript-project-reconcile.mjs";

const repo=path.resolve(new URL("../..",import.meta.url).pathname),fixture=path.join(repo,"fixtures/javascript-upb05-configuration"),projectPath="example.test/javascript-upb05",files=["application.js","configuration.js","controlled.js","policy.js","state.js","transport.js"];
const options=(root,out,replacement="BuildPolicy")=>({project:root,destination:out,projectPath,files,moduleG1:path.join(repo,"modules/execution/v36/module.g1"),entryName:"Run",target:javascriptDeclarationIdentity(projectPath,"InitializePolicy"),expected:"InitializePolicy",replacement,revision:2,nativeRunner:path.join(repo,"reference/js/javascript-upb11-native-runner.mjs")});

test("performs an identity-bound cross-module rename atomically",()=>{const work=fs.mkdtempSync(path.join(os.tmpdir(),"seme-js-reconcile-test-"));try{const out=path.join(work,"out");const result=reconcileJavaScriptProject(options(fixture,out));assert.equal(result.report.occurrences.length,2);assert.match(fs.readFileSync(path.join(out,"policy.js"),"utf8"),/function BuildPolicy/);assert.match(fs.readFileSync(path.join(out,"application.js"),"utf8"),/import \{ BuildPolicy \}/);assert.doesNotMatch(fs.readFileSync(path.join(out,"application.js"),"utf8"),/InitializePolicy/);assert.equal(result.report.target,options(fixture,out).target);assert.throws(()=>reconcileJavaScriptProject(options(fixture,out)),/javascript_reconcile\.path/);}finally{fs.rmSync(work,{recursive:true,force:true});}});
test("rejects forged identities and collisions without output",()=>{const work=fs.mkdtempSync(path.join(os.tmpdir(),"seme-js-reconcile-test-"));try{assert.throws(()=>reconcileJavaScriptProject({...options(fixture,path.join(work,"forged")),target:"00".repeat(16)}),/target_identity/);assert.throws(()=>reconcileJavaScriptProject(options(fixture,path.join(work,"collision"),"Run")),/collision/);assert.equal(fs.existsSync(path.join(work,"forged")),false);assert.equal(fs.existsSync(path.join(work,"collision")),false);}finally{fs.rmSync(work,{recursive:true,force:true});}});
