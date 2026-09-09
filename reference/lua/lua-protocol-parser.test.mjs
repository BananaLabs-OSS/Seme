import test from "node:test";
import assert from "node:assert/strict";
import { parseProtocolDeclarations, stripProtocolDeclarations } from "./lua-protocol-parser.mjs";

test("parses closed protocols and exact implementation witnesses", () => {
  const source = `local P = Seme.protocol("Adjuster", { "adjust" })
local W = Seme.implementation(P, "record", {
  adjust = Offset_adjust,
})`;
  const parsed = parseProtocolDeclarations(source);
  assert.deepEqual(parsed.protocols.get("P").requirements, ["adjust"]);
  assert.equal(parsed.implementations.get("W").methods.get("adjust"), "Offset_adjust");
  assert.doesNotMatch(stripProtocolDeclarations(source), /Seme\.(?:protocol|implementation)/);
});

for (const [name, source, code] of [
  ["duplicates", `local P = Seme.protocol("P", { "x", "x" })`, "lua.protocol_requirements"],
  ["missing", `local P = Seme.protocol("P", { "x" })\nlocal W = Seme.implementation(P, "record", {\n})`, "lua.implementation_method_set"],
  ["extra", `local P = Seme.protocol("P", { "x" })\nlocal W = Seme.implementation(P, "record", {\n x = f, extra = g,\n})`, "lua.implementation_method_set"],
  ["unknown protocol", `local W = Seme.implementation(P, "record", {\n x = f,\n})`, "lua.implementation_protocol_scope"],
  ["monkey patch", `local P = Seme.protocol("P", { "x" })\nP.x = f`, "lua.protocol_monkey_patch"],
]) test(`rejects ${name}`, () => assert.throws(() => parseProtocolDeclarations(source), new RegExp(code)));
