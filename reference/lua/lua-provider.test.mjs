import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftLua } from "./lua-provider.mjs";
import { projectLua } from "./lua-projector.mjs";

const scopedTypeProgram = fs.readFileSync(new URL("../../fixtures/go-execution-v81/program.g1", import.meta.url), "utf8");

test("projects scoped semantic types through an explicit Lua adapter", () => {
  const projected = projectLua(scopedTypeProgram);
  assert.match(projected, /Seme\.scoped_type\(/);
  assert.doesNotMatch(projected, /---@class local/);
});

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
const moduleV31G1 = fs.readFileSync(new URL("../../modules/execution/v31/module.g1", import.meta.url), "utf8");
const moduleV33G1 = fs.readFileSync(new URL("../../modules/execution/v33/module.g1", import.meta.url), "utf8");
const sources = [
  { name: "identity.lua", source: `---@param value boolean\n---@return boolean\nlocal function Identity(value)\n  return value\nend\n` },
  { name: "run.lua", source: `---@param enabled boolean\n---@return boolean\nfunction Run(enabled)\n  return Identity(enabled)\nend\n` },
];

test("lifts multiple Lua files and calls directly into canonical Seme", () => {
  const options = { sources, moduleG1, packagePath: "example.test/lua-uab-01", revision: 1, entryName: "Run" };
  assert.equal(liftLua(options), liftLua(options));
  assert.match(liftLua(options), /00000000000000000000000000009060/);
});

test("projects idiomatic bounded Lua and re-lifts byte-identically", () => {
  const options = { sources, moduleG1, packagePath: "example.test/lua-uab-01", revision: 1, entryName: "Run" };
  const canonical = liftLua(options);
  const projected = projectLua(canonical);
  assert.match(projected, /local function Identity/);
  assert.match(projected, /function Run/);
  const relifted = liftLua({ sources: [{ name: "projected.lua", source: projected }], moduleG1, packagePath: options.packagePath, revision: 1, entryName: "Run" });
  assert.equal(relifted, canonical);
});

test("rejects nearby Lua semantics with located diagnostics", () => {
  const base = { moduleG1, packagePath: "example.test/lua-reject", revision: 1, entryName: "Run" };
  assert.throws(() => liftLua({ ...base, sources: [{ name: "bad.lua", source: "function Run(value)\n return value\nend" }] }), /lua\.unsupported_annotation:bad\.lua:1:1/);
  assert.throws(() => liftLua({ ...base, sources: [{ name: "bad.lua", source: "---@param value boolean\n---@return boolean\nfunction Run(value)\n return Missing(value)\nend" }] }), /lua\.unknown_call:bad\.lua:4:1/);
  assert.throws(() => liftLua({ ...base, sources: [{ name: "bad.lua", source: "---@param value boolean\n---@return boolean\nfunction Run(value)\n value = false\n return value\nend" }] }), /lua\.assignment_scope:bad\.lua:4:1/);
});

test("projects Go-owned map range through an explicit Lua adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v70/program.g1", import.meta.url), "utf8");
  const projected = projectLua(canonical);
  assert.match(projected, /Seme\.native_range\("go"/);
  assert.match(projected, /value/);
});

test("projects Go-owned slicing through an explicit Lua adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v71/program.g1", import.meta.url), "utf8");
  assert.match(projectLua(canonical), /Seme\.native_slice\("go"/);
});

test("projects Go-owned pointer dereference through an explicit Lua adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v72/program.g1", import.meta.url), "utf8");
  assert.match(projectLua(canonical), /Seme\.native_dereference\("go"/);
});

test("projects Go-owned binary operators through explicit Lua adapters", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v73/program.g1", import.meta.url), "utf8");
  const projected = projectLua(canonical);
  assert.match(projected, /Seme\.native_binary\("go", "<<"/);
  assert.match(projected, /Seme\.native_binary\("go", "\|"/);
});

test("projects Go-owned indexed assignment through an explicit Lua adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v74/program.g1", import.meta.url), "utf8");
  assert.match(projectLua(canonical), /Seme\.assign_native_index\("go"/);
});

test("lifts typed UAB-03 blocks and projects them byte-identically", () => {
  const source = fs.readFileSync(new URL("../../fixtures/lua-uab-03/program.lua", import.meta.url), "utf8");
  const options = { sources:[{name:"program.lua",source}], moduleG1, packagePath:"example.test/lua-uab03", revision:1, entryName:"Accumulate" };
  const canonical=liftLua(options),projected=projectLua(canonical),relifted=liftLua({...options,sources:[{name:"projected.lua",source:projected}]});
  assert.equal(relifted,canonical); for(const suffix of [0x90e0,0x90e1,0x90e2,0x90e3,0x90e4,0x90f0,0x9021,0x90b1])assert.match(canonical,new RegExp(schemaID(suffix)));
});

test("keeps the four explicit scalar boundaries distinct", () => {
  for (const type of ["boolean", "seme.i64", "seme.text", "seme.bytes"]) {
    const source = `---@param value ${type}\n---@return ${type}\nfunction Identity(value)\n  return value\nend\n`;
    const options = { sources: [{ name: `${type}.lua`, source }], moduleG1, packagePath: `example.test/lua-${type}`, revision: 1, entryName: "Identity" };
    const canonical = liftLua(options);
    const projected = projectLua(canonical);
    assert.match(projected, new RegExp(type.replace(".", "\\.")));
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
  }
});

test("rejects scalar type conflation", () => {
  const source = `---@param value seme.bytes\n---@return seme.text\nfunction Decode(value)\n  return value\nend\n`;
  assert.throws(() => liftLua({ sources: [{ name: "decode.lua", source }], moduleG1, packagePath: "example.test/lua-types", revision: 1, entryName: "Decode" }), /lua\.return_type:decode\.lua:4:1/);
});

test("lifts and projects explicit Option constructors through Core v31", () => {
  for (const [name, parameters, expression] of [["Some", "---@param value seme.i64\n", "Seme.some(value)"], ["None", "", "Seme.none()"]]) {
    const source = `${parameters}---@return seme.option<seme.i64>\nfunction ${name}(${parameters ? "value" : ""})\n  return ${expression}\nend\n`;
    const options = { sources: [{ name: `${name}.lua`, source }], moduleG1: moduleV31G1, packagePath: `example.test/lua-option-${name}`, revision: 1, entryName: name };
    const canonical = liftLua(options);
    assert.match(canonical, new RegExp(name === "Some" ? schemaID(0xa052) : schemaID(0xa051)));
    const projected = projectLua(canonical);
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
  }
});

test("lifts and projects explicit Result arms", () => {
  for (const [name, parameterType, expression] of [["Accept", "seme.i64", "Seme.ok(value)"], ["Reject", "seme.text", "Seme.err(value)"]]) {
    const source = `---@param value ${parameterType}\n---@return seme.result<seme.i64,seme.text>\nfunction ${name}(value)\n  return ${expression}\nend\n`;
    const options = { sources: [{ name: `${name}.lua`, source }], moduleG1: moduleV31G1, packagePath: `example.test/lua-result-${name}`, revision: 1, entryName: name };
    const canonical = liftLua(options);
    const projected = projectLua(canonical);
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
  }
});

test("rejects nil and wrong typed tagged values", () => {
  const base = { moduleG1: moduleV31G1, packagePath: "example.test/lua-tagged-reject", revision: 1, entryName: "Run" };
  assert.throws(() => liftLua({ ...base, sources: [{ name: "nil.lua", source: "---@return seme.option<seme.i64>\nfunction Run()\n return nil\nend" }] }), /lua\.unknown_identifier:nil\.lua:3:1/);
  assert.throws(() => liftLua({ ...base, sources: [{ name: "wrong.lua", source: "---@param value seme.text\n---@return seme.option<seme.i64>\nfunction Run(value)\n return Seme.some(value)\nend" }] }), /lua\.constructor_value_type:wrong\.lua:4:1/);
});

test("maps explicit fixed-array, slice, and runtime-map wrapper types", () => {
  for (const [name, type, schema] of [
    ["Array", "seme.array<seme.i64,3>", 0x90f2],
    ["Slice", "seme.slice<seme.text>", 0x90f8],
    ["Map", "seme.map<seme.i64,seme.bytes>", 0xa040],
  ]) {
    const source = `---@param value ${type}\n---@return ${type}\nfunction ${name}(value)\n  return value\nend\n`;
    const options = { sources: [{ name: `${name}.lua`, source }], moduleG1: moduleV31G1, packagePath: `example.test/lua-${name}`, revision: 1, entryName: name };
    const canonical = liftLua(options);
    assert.match(canonical, new RegExp(schemaID(schema)));
    const projected = projectLua(canonical);
    assert.match(projected, new RegExp(type.replace(/[.<>]/g, "\\$&")));
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
  }
});

test("rejects ambiguous raw table collection annotations", () => {
  const source = `---@param value table\n---@return table\nfunction Raw(value)\n  return value\nend\n`;
  assert.throws(() => liftLua({ sources: [{ name: "raw.lua", source }], moduleG1: moduleV31G1, packagePath: "example.test/lua-raw", revision: 1, entryName: "Raw" }), /lua\.unsupported_annotation:raw\.lua:3:1/);
});

test("maps declared ordered records without inferring raw table shape", () => {
  const source = `---@class Bounds\n---@field minimum seme.i64\n---@field maximum seme.i64\n\n---@param value Bounds\n---@return Bounds\nfunction IdentityBounds(value)\n  return value\nend\n`;
  const options = { sources: [{ name: "bounds.lua", source }], moduleG1: moduleV31G1, packagePath: "example.test/lua-record", revision: 1, entryName: "IdentityBounds" };
  const canonical = liftLua(options);
  assert.match(canonical, new RegExp(schemaID(0x9030)));
  const projected = projectLua(canonical);
  assert.match(projected, /---@class Bounds\n---@field minimum seme\.i64\n---@field maximum seme\.i64/);
  assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
});

test("maps explicit record field reads", () => {
  const source = `---@class Bounds\n---@field minimum seme.i64\n---@field maximum seme.i64\n\n---@param value Bounds\n---@return seme.i64\nfunction Minimum(value)\n  return Seme.field(value, "minimum")\nend\n`;
  const options = { sources: [{ name: "field.lua", source }], moduleG1: moduleV31G1, packagePath: "example.test/lua-field", revision: 1, entryName: "Minimum" };
  const canonical = liftLua(options);
  assert.match(canonical, new RegExp(schemaID(0x9032)));
  const projected = projectLua(canonical);
  assert.match(projected, /Seme\.field\(value, "minimum"\)/);
  assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
});

test("maps explicit array construction and zero-based queries", () => {
  const cases = [
    ["Build", "---@param a seme.i64\n---@param b seme.i64\n---@return seme.array<seme.i64,2>", "a, b", "Seme.array(a, b)", 0x90f3],
    ["Length", "---@param values seme.array<seme.i64,2>\n---@return seme.i64", "values", "Seme.length(values)", 0x90f9],
    ["First", "---@param values seme.array<seme.i64,2>\n---@return seme.i64", "values", "Seme.index_zero(values, 0)", 0x90f4],
  ];
  for (const [name, docs, parameters, expression, expected] of cases) {
    const source = `${docs}\nfunction ${name}(${parameters})\n  return ${expression}\nend\n`;
    const options = { sources: [{ name: `${name}.lua`, source }], moduleG1: moduleV31G1, packagePath: `example.test/lua-${name}`, revision: 1, entryName: name };
    const canonical = liftLua(options); assert.match(canonical, new RegExp(schemaID(expected)));
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projectLua(canonical) }] }), canonical);
  }
});

test("maps explicit empty-map, zero lookup, and immutable update", () => {
  const cases = [
    ["Empty", "---@return seme.map<seme.i64,seme.i64>", "", "Seme.empty_map(\"i64\")", 0xa041],
    ["Lookup", "---@param values seme.map<seme.i64,seme.i64>\n---@param key seme.i64\n---@return seme.i64", "values, key", "Seme.lookup_zero(values, key, \"i64\")", 0xa042],
    ["Update", "---@param values seme.map<seme.i64,seme.i64>\n---@param key seme.i64\n---@param value seme.i64\n---@return seme.map<seme.i64,seme.i64>", "values, key, value", "Seme.map_update(values, key, value)", 0xa043],
  ];
  for (const [name, docs, parameters, expression, expected] of cases) {
    const source = `${docs}\nfunction ${name}(${parameters})\n  return ${expression}\nend\n`;
    const options = { sources: [{ name: `${name}.lua`, source }], moduleG1: moduleV31G1, packagePath: `example.test/lua-${name}`, revision: 1, entryName: name };
    const canonical = liftLua(options); assert.match(canonical, new RegExp(schemaID(expected)));
    assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projectLua(canonical) }] }), canonical);
  }
});

test("rejects a map zero descriptor that disagrees with its value type", () => {
  const source = `---@return seme.map<seme.i64,seme.text>\nfunction Bad()\n return Seme.empty_map("bytes")\nend`;
  assert.throws(() => liftLua({ sources: [{ name: "bad-map.lua", source }], moduleG1: moduleV31G1, packagePath: "example.test/lua-map-descriptor", revision: 1, entryName: "Bad" }), /lua\.map_descriptor_type:bad-map\.lua:3:1/);
});

test("rejects one-based or out-of-range canonical indexes", () => {
  const source = `---@param values seme.array<seme.i64,2>\n---@return seme.i64\nfunction Bad(values)\n return Seme.index_zero(values, 2)\nend`;
  assert.throws(() => liftLua({ sources: [{ name: "bad-index.lua", source }], moduleG1: moduleV31G1, packagePath: "example.test/lua-index", revision: 1, entryName: "Bad" }), /lua\.index_type_or_range:bad-index\.lua:4:1/);
});

test("lifts and projects structural total Option/Result/bytes matching", () => {
  const expression = 'Seme.match_option(value, false, function(some) return Seme.match_result(some, function(ok) return Seme.bytes_equal(ok, Seme.bytes_literal("ok")) end, function(err) return Seme.text_equal(err, "bad") end) end)';
  const source = `---@param value seme.option<seme.result<seme.bytes,seme.text>>\n---@return boolean\nfunction Check(value)\n  return ${expression}\nend\n`;
  const options = { sources: [{ name: "match.lua", source }], moduleG1: fs.readFileSync(new URL("../../modules/execution/v32/module.g1", import.meta.url), "utf8"), packagePath: "example.test/lua-composite", revision: 1, entryName: "Check" };
  const canonical = liftLua(options);
  for (const suffix of [0xa060, 0xa061, 0xa062, 0xa063, 0xa064, 0xa065]) assert.match(canonical, new RegExp(schemaID(suffix)));
  const projected = projectLua(canonical);
  assert.match(projected, /Seme\.match_option/);
  assert.equal(liftLua({ ...options, sources: [{ name: "projected.lua", source: projected }] }), canonical);
});

test("rejects non-total or unscoped composite matching", () => {
  const module = fs.readFileSync(new URL("../../modules/execution/v32/module.g1", import.meta.url), "utf8");
  const source = `---@param value seme.option<seme.result<seme.bytes,seme.text>>\n---@return boolean\nfunction Check(value)\n  return Seme.match_option(value, false, function(some) return true end)\nend`;
  assert.throws(() => liftLua({ sources: [{ name: "partial.lua", source }], moduleG1: module, packagePath: "example.test/lua-partial", revision: 1, entryName: "Check" }), /lua\.match_arm_expression:partial\.lua:4:1/);
});

test("lifts and projects explicit immutable v33 collection mutations", () => {
  const cases=[
    ["Construct","---@param first seme.i64\n---@param second seme.i64\n---@return seme.slice<seme.i64>","Seme.slice(first, second)",0xa068],
    ["Append","---@param values seme.slice<seme.i64>\n---@param value seme.i64\n---@return seme.slice<seme.i64>","Seme.collection_append(values, value)",0x90fb],
    ["Update","---@param values seme.slice<seme.i64>\n---@param index seme.i64\n---@param value seme.i64\n---@return seme.slice<seme.i64>","Seme.collection_update(values, index, value)",0x90fc],
    ["Remove","---@param values seme.slice<seme.i64>\n---@param index seme.i64\n---@return seme.slice<seme.i64>","Seme.slice_remove(values, index)",0xa066],
    ["MapRemove","---@param values seme.map<seme.i64,seme.i64>\n---@param key seme.i64\n---@return seme.map<seme.i64,seme.i64>","Seme.map_remove(values, key)",0xa067],
    ["Fold","---@param values seme.slice<seme.i64>\n---@param initial seme.i64\n---@return seme.i64","Seme.fold(values, initial, function(accumulator, element) return Seme.add(accumulator, element) end)",0x90f7],
  ];
  for(const [name,annotations,expression,suffix] of cases){const source=`${annotations}\nfunction ${name}(${annotations.split("\n").filter(line=>line.startsWith("---@param")).map(line=>line.split(/\s+/)[1]).join(", ")})\n return ${expression}\nend`;const options={sources:[{name:"v33.lua",source}],moduleG1:moduleV33G1,packagePath:`example.test/lua-v33-${name}`,revision:1,entryName:name};const canonical=liftLua(options),projected=projectLua(canonical);assert.match(canonical,new RegExp(schemaID(suffix)));assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);}
});

test("projects every entry in the shared Lua UAB-04 corpus",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-04/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v34/module.g1",import.meta.url),"utf8");for(const entryName of ["ConstructSlice","Append","Update","Remove","Length","Index","Traverse","EmptyMap","Insert","Lookup","RemoveMap"]){const options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:`example.test/lua-uab04-${entryName}`,revision:1,entryName};const canonical=liftLua(options),projected=projectLua(canonical);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);}});

test("recursively types nested collection expressions",()=>{const source=`---@param first seme.i64\n---@param second seme.i64\n---@return seme.i64\nfunction Nested(first, second)\n return Seme.length(Seme.slice(first, second))\nend`;const options={sources:[{name:"nested.lua",source}],moduleG1:fs.readFileSync(new URL("../../modules/execution/v34/module.g1",import.meta.url),"utf8"),packagePath:"example.test/lua-nested",revision:1,entryName:"Nested"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.length\(Seme\.slice\(first, second\)\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts explicit Lua protocols to neutral methods, witnesses, and dynamic dispatch",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-05/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v27/module.g1",import.meta.url),"utf8"),options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-05",revision:1,entryName:"Dispatch"},canonical=liftLua(options),projected=projectLua(canonical);for(const suffix of [0xa000,0xa001,0xa002,0xa010,0xa011,0xa012,0xa013,0xa014])assert.match(canonical,new RegExp(schemaID(suffix)));assert.doesNotMatch(canonical,/by 4f66667365745f61646a757374\n.*00000000000000000000000000009011/s);assert.match(projected,/Seme\.protocol_dispatch/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts explicit Lua immutable and mutable closure adapters and projects byte-identically",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-06/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v29/module.g1",import.meta.url),"utf8");for(const [entry,suffixes]of [["Immutable",[0xa020,0xa021,0xa022,0xa023,0xa024]],["Mutable",[0xa020,0xa030,0xa031,0xa032,0xa033,0xa034,0xa035]]]){const options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-06",revision:1,entryName:entry},canonical=liftLua(options),projected=projectLua(canonical);for(const suffix of suffixes)assert.match(canonical,new RegExp(schemaID(suffix)));assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);}});

test("rejects raw Lua closures outside the explicit neutral adapter",()=>{const source=`---@param base seme.i64\n---@return seme.i64\nfunction Bad(base)\n return (function(value) return Seme.add(base, value) end)(base)\nend`;assert.throws(()=>liftLua({sources:[{name:"raw-closure.lua",source}],moduleG1:fs.readFileSync(new URL("../../modules/execution/v29/module.g1",import.meta.url),"utf8"),packagePath:"bad",revision:1,entryName:"Bad"}),/lua\.unsupported_expression:raw-closure\.lua:4:1/);});

test("lifts sealed typed Lua transitions and preserves state/result distinction",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-07/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v27/module.g1",import.meta.url),"utf8"),options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-07",revision:1,entryName:"Step"},canonical=liftLua(options),projected=projectLua(canonical);for(const suffix of [0xa004,0xa005])assert.match(canonical,new RegExp(schemaID(suffix)));assert.match(projected,/Seme\.transition_step/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts total sealed Result propagation and re-lifts byte-identically",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-08/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v32/module.g1",import.meta.url),"utf8"),options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-08",revision:1,entryName:"IncrementPositive"},canonical=liftLua(options),projected=projectLua(canonical);for(const suffix of [0x9042,0x9043,0x9044,0xa060,0xa061,0xa062])assert.match(canonical,new RegExp(schemaID(suffix)));assert.match(projected,/Seme\.increment_positive/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts ordered authorized Lua observations and projects byte-identically",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-09/program.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v20/module.g1",import.meta.url),"utf8"),options={sources:[{name:"program.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-09",revision:1,entryName:"Observe"},canonical=liftLua(options),projected=projectLua(canonical),effectEntities=canonical.match(new RegExp(`^en (?!${schemaID(0x90f1)} )[0-9a-f]{32} ${schemaID(0x90f1)} `,"gm"))||[];assert.equal(effectEntities.length,2);assert.match(canonical,/6f62736572766162696c6974792e6c6f67/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("preserves explicit semantic identity through formatting and rejects forged identities",()=>{const module=fs.readFileSync(new URL("../../modules/execution/v30/module.g1",import.meta.url),"utf8"),identity="80112233445566778899aabbccddeeff",plain=`---@seme-id ${identity}\n---@param value boolean\n---@return boolean\nfunction Identity(value)\n return value\nend\n`,formatted=`\n---@seme-id   ${identity}\n---@param value boolean\n---@return boolean\nfunction   Identity ( value )\n\n  return value\nend\n`,options={moduleG1:module,packagePath:"example.test/lua-uab-10",revision:1,entryName:"Identity"};assert.equal(liftLua({...options,sources:[{name:"plain.lua",source:plain}]}),liftLua({...options,sources:[{name:"formatted.lua",source:formatted}]}));assert.throws(()=>liftLua({...options,sources:[{name:"bad.lua",source:plain.replace(identity,"00000000000000000000000000009011")}]}),/lua\.invalid_semantic_identity:bad\.lua:1:1/);const duplicate=`${plain}\n---@seme-id ${identity}\n---@param value boolean\n---@return boolean\nlocal function Other(value)\n return value\nend\n`;assert.throws(()=>liftLua({...options,sources:[{name:"duplicate.lua",source:duplicate}]}),/lua\.duplicate_semantic_identity:duplicate\.lua:[1-9][0-9]*:/);});

test("lifts and projects neutral modular i64 multiplication",()=>{const source=`---@param left seme.i64\n---@param right seme.i64\n---@return seme.i64\nfunction Multiply(left, right)\n return Seme.multiply(left, right)\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v34/module.g1",import.meta.url),"utf8"),options={sources:[{name:"multiply.lua",source}],moduleG1:module,packagePath:"example.test/lua-multiply",revision:1,entryName:"Multiply"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(canonical,new RegExp(schemaID(0x9090)));assert.match(projected,/Seme\.multiply\(left, right\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts Option-returning runtime map lookup through the neutral v35 schema",()=>{const source=`---@param values seme.map<seme.i64,seme.i64>\n---@param key seme.i64\n---@return seme.option<seme.i64>\nfunction Lookup(values, key)\n return Seme.map_lookup(values, key)\nend\n`,v35=new URL("../../modules/execution/v35/module.g1",import.meta.url),module=fs.readFileSync(fs.existsSync(v35)?v35:new URL("../../modules/execution/v34/module.g1",import.meta.url),"utf8"),options={sources:[{name:"lookup.lua",source}],moduleG1:module,packagePath:"example.test/lua-map-lookup-option",revision:1,entryName:"Lookup"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(canonical,new RegExp(schemaID(0xa044)));assert.match(projected,/Seme\.map_lookup\(values, key\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);const wrong=source.replace("seme.option<seme.i64>","seme.i64");assert.throws(()=>liftLua({...options,sources:[{name:"wrong.lua",source:wrong}]}),/lua\.map_lookup_option_type:wrong\.lua:5:1/);});

test("lifts scalar protocol witnesses and runtime-selected dynamic dispatch",()=>{const source=fs.readFileSync(new URL("../../fixtures/lua-uab-11/policy.lua",import.meta.url),"utf8"),module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"policy.lua",source}],moduleG1:module,packagePath:"example.test/lua-uab-11-policy",revision:1,entryName:"ApplyPolicy"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.protocol_dispatch_value/);assert.match(projected,/Seme\.multiply/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("composes nested record field reads into Option-returning map lookup",()=>{const source=`---@class State\n---@field Counters seme.map<seme.i64,seme.i64>\n\n---@class Command\n---@field Key seme.i64\n\n---@param state State\n---@param command Command\n---@return seme.option<seme.i64>\nfunction LookupCounter(state, command)\n return Seme.map_lookup(Seme.field(state, "Counters"), Seme.field(command, "Key"))\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"nested.lua",source}],moduleG1:module,packagePath:"example.test/lua-nested-map",revision:1,entryName:"LookupCounter"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.map_lookup\(Seme\.field\(state, "Counters"\), Seme\.field\(command, "Key"\)\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts generic total Option callback arms with scoped payload binding",()=>{const source=`---@param candidate seme.option<seme.i64>\n---@param fallback seme.i64\n---@return seme.i64\nfunction Resolve(candidate, fallback)\n return Seme.match_option(candidate, fallback, function(value) return Seme.add(value, fallback) end)\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"option.lua",source}],moduleG1:module,packagePath:"example.test/lua-option-callback",revision:1,entryName:"Resolve"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.match_option\(candidate, fallback, function\(value\)[\s\S]*Seme\.add\(value, fallback\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);const bad=source.replace("Seme.add(value, fallback)","candidate");assert.throws(()=>liftLua({...options,sources:[{name:"bad.lua",source:bad}]}),/lua\.collection_argument_type|lua\.match_arm_type/);});

test("lifts generic total Result callback arms with independently scoped payloads",()=>{const source=`---@param candidate seme.result<seme.i64,seme.i64>\n---@param amount seme.i64\n---@return seme.i64\nfunction ResolveResult(candidate, amount)\n return Seme.match_result(candidate, function(value) return Seme.add(value, amount) end, function(error) return error end)\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"result.lua",source}],moduleG1:module,packagePath:"example.test/lua-result-callback",revision:1,entryName:"ResolveResult"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.match_result\(candidate, function\(value\)[\s\S]*Seme\.add\(value, amount\)[\s\S]*function\(error\)[\s\S]*return error/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts traversal with mutable accumulation and nested early return",()=>{const source=`---@param values seme.slice<seme.i64>\n---@param last seme.i64\n---@return seme.i64\nfunction Scan(values, last)\n local index = Seme.i64_literal("0")\n local total = Seme.i64_literal("0")\n while Seme.less_equal(index, last) do\n  if Seme.less_equal(Seme.index_zero(values, index), Seme.i64_literal("-1")) then\n   return total\n  end\n  total = Seme.add(total, Seme.index_zero(values, index))\n  index = Seme.add(index, Seme.i64_literal("1"))\n end\n return total\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"scan.lua",source}],moduleG1:module,packagePath:"example.test/lua-scan",revision:1,entryName:"Scan"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/while Seme\.less_equal/);assert.match(projected,/Seme\.i64_literal\("-1"\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts nested Result constructors from early control-block returns",()=>{const source=`---@param invalid boolean\n---@param value seme.i64\n---@return seme.result<seme.i64,seme.i64>\nfunction Validate(invalid, value)\n if Seme.boolean(invalid) then\n  return Seme.err(Seme.i64_literal("1"))\n end\n return Seme.ok(value)\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"validate.lua",source}],moduleG1:module,packagePath:"example.test/lua-result-control",revision:1,entryName:"Validate"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/return Seme\.err\(Seme\.i64_literal\("1"\)\)/);assert.match(projected,/return Seme\.ok\(value\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts ordered generic record construction from typed control expressions",()=>{const source=`---@class Pair\n---@field Name seme.text\n---@field Value seme.i64\n\n---@param name seme.text\n---@param value seme.i64\n---@return Pair\nfunction MakePair(name, value)\n return Seme.record("Pair", { "Name", "Value" }, { Name = name, Value = value })\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"record.lua",source}],moduleG1:module,packagePath:"example.test/lua-record-construct",revision:1,entryName:"MakePair"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.record\("Pair"/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);const swapped=source.replace('{ Name = name, Value = value }','{ Value = value, Name = name }');assert.throws(()=>liftLua({...options,sources:[{name:"swapped.lua",source:swapped}]}),/lua\.record_construct_fields:swapped\.lua:9:1/);});

test("projects nested record-transition-Result construction without the direct-transition shortcut",()=>{const source=`---@class Pair\n---@field Name seme.text\n---@field Value seme.i64\n\n---@param name seme.text\n---@param value seme.i64\n---@return seme.result<seme.transition<Pair,seme.i64>,seme.i64>\nfunction Commit(name, value)\n return Seme.ok(Seme.transition(Seme.record("Pair", { "Name", "Value" }, { Name = name, Value = value }), value))\nend\n`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"commit.lua",source}],moduleG1:module,packagePath:"example.test/lua-nested-transition",revision:1,entryName:"Commit"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(projected,/Seme\.ok\(Seme\.transition\(Seme\.record/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("lifts generic multiline Option callbacks as typed blocks",()=>{const source=`---@param candidate seme.option<seme.i64>
---@param fallback seme.i64
---@return seme.i64
function ResolveBlock(candidate, fallback)
 return Seme.match_option(candidate, fallback, function(value)
  local adjusted = Seme.add(value, fallback)
  return adjusted
 end)
end
`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"option-block.lua",source}],moduleG1:module,packagePath:"example.test/lua-option-block",revision:1,entryName:"ResolveBlock"},canonical=liftLua(options);assert.match(canonical,new RegExp(schemaID(0xa063)));assert.match(canonical,new RegExp(schemaID(0xa061)));const escaped=source.replace("Seme.add(value, fallback)","Seme.add(candidate, fallback)");assert.throws(()=>liftLua({...options,sources:[{name:"escaped.lua",source:escaped}]}),/lua\.control_expression_type|lua\.add_result_type/);});

test("lifts, projects, and byte-relifts the cumulative Lua UAB-11 application",()=>{const names=["policy.lua","application.lua"],sources=names.map(name=>({name,source:fs.readFileSync(new URL(`../../fixtures/lua-uab-11/${name}`,import.meta.url),"utf8")})),moduleG1=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources,moduleG1,packagePath:"example.test/lua-uab-11",revision:1,entryName:"Apply"},canonical=liftLua(options),projected=projectLua(canonical);assert.match(canonical,new RegExp(schemaID(0xa034)));assert.match(canonical,new RegExp(schemaID(0xa035)));assert.match(canonical,new RegExp(schemaID(0xa063)));assert.match(canonical,new RegExp(schemaID(0x90f1)));assert.match(projected,/local accumulate = Seme\.mutable_closure\(total, function\(total, value\) return Seme\.add\(total, value\) end\)/);assert.match(projected,/local adjusted = add_amount\(selected\)/);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);});

test("accepts only an intact self-verifying Lua projection envelope",()=>{const source=`---@param value seme.i64
---@return seme.i64
function Increment(value)
 return Seme.add(value, Seme.i64_literal("1"))
end
`,module=fs.readFileSync(new URL("../../modules/execution/v35/module.g1",import.meta.url),"utf8"),options={sources:[{name:"source.lua",source}],moduleG1:module,packagePath:"example.test/lua-envelope",revision:1,entryName:"Increment"},canonical=liftLua(options),projected=projectLua(canonical);assert.equal(liftLua({...options,sources:[{name:"projected.lua",source:projected}]}),canonical);assert.throws(()=>liftLua({...options,sources:[{name:"edited.lua",source:projected.replace('Seme.i64_literal("1")','Seme.i64_literal("2")')}]}),/lua_projection\.envelope_source_mismatch/);assert.throws(()=>liftLua({...options,sources:[{name:"digest.lua",source:projected.replace(/projection-v1 [0-9a-f]/,"projection-v1 0")}]}),/lua_projection\.envelope_digest/);});

function schemaID(suffix) { return suffix.toString(16).padStart(32, "0"); }
