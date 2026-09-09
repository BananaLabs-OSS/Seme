import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftLua } from "./lua-provider.mjs";
import { projectLua } from "./lua-projector.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
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
  assert.throws(() => liftLua({ ...base, sources: [{ name: "bad.lua", source: "---@param value boolean\n---@return boolean\nfunction Run(value)\n value = false\n return value\nend" }] }), /lua\.function_body_profile:bad\.lua:3:1/);
});
