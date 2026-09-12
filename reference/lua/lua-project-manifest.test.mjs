import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import test from "node:test";
import {liftLua} from "./lua-provider.mjs";
import {emitLuaProjectManifest} from "./lua-project-manifest.mjs";

const repo=path.resolve(new URL("../..",import.meta.url).pathname),identity="seme.uab11/application";
test("derives the entry signature and owned effects deterministically",()=>{const canonical=liftLua({sources:["policy.lua","application.lua"].map((name)=>({name:path.join(repo,"fixtures/lua-uab-11",name),source:fs.readFileSync(path.join(repo,"fixtures/lua-uab-11",name),"utf8")})),moduleG1:fs.readFileSync(path.join(repo,"modules/execution/v35/module.g1"),"utf8"),packagePath:identity,revision:1,entryName:"Apply"}),a=emitLuaProjectManifest(canonical,identity),b=emitLuaProjectManifest(canonical,identity);assert.deepEqual(a,b);assert.equal(a.packages[0].interfaces[0].name,"Apply");assert.equal(a.packages[0].interfaces[0].parameters.length,2);assert.equal(a.packages[0].effects.length,1);});
test("rejects malformed and ambiguous graphs",()=>{assert.throws(()=>emitLuaProjectManifest("",identity),/program_cardinality/);assert.throws(()=>emitLuaProjectManifest("en broken",identity),/entity/);});
