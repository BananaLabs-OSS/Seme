import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import {LuaProjectSession} from "./lua-project-reconcile.mjs";
const moduleG1=fs.readFileSync(new URL("../../modules/execution/v36/module.g1",import.meta.url),"utf8"),names=["application.lua","configuration.lua","controlled.lua","policy.lua","state.lua","transport.lua"],files=names.map(name=>({name,source:fs.readFileSync(new URL(`../../fixtures/lua-upb05-configuration/${name}`,import.meta.url),"utf8")}));
test("session preserves last valid graph across invalid and stale snapshots",()=>{const session=new LuaProjectSession({projectPath:"example.test/lua-upb05",moduleG1,entryName:"Run"}),valid=session.apply({revision:1,files});assert.equal(valid.disposition,"accepted-valid");const invalid=session.apply({revision:2,files:files.map(file=>file.name==="application.lua"?{...file,source:"not lua !!!"}:file)});assert.equal(invalid.disposition,"accepted-invalid");assert.equal(invalid.canonicalG1,valid.canonicalG1);assert.equal(session.apply({revision:1,files}).disposition,"rejected-stale");});
