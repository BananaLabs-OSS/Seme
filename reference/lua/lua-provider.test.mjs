import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftLua } from "./lua-provider.mjs";
import { projectLua } from "./lua-projector.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
const moduleV31G1 = fs.readFileSync(new URL("../../modules/execution/v31/module.g1", import.meta.url), "utf8");
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
  assert.throws(() => liftLua({ sources: [{ name: "partial.lua", source }], moduleG1: module, packagePath: "example.test/lua-partial", revision: 1, entryName: "Check" }), /lua\.unsupported_expression:partial\.lua:4:1/);
});

function schemaID(suffix) { return suffix.toString(16).padStart(32, "0"); }
