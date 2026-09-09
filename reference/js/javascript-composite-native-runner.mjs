import { Seme } from "./seme-values.mjs";

globalThis.Seme = Seme;
const module = await import(process.argv[2] || new URL("../../fixtures/javascript-uab-02/composite.js", import.meta.url));
const cases = [
  [Seme.none(), false],
  [Seme.some(Seme.ok(Seme.bytes([111, 107]))), true],
  [Seme.some(Seme.ok(Seme.bytes([110, 111]))), false],
  [Seme.some(Seme.error("bad")), true],
  [Seme.some(Seme.error("no")), false],
];
const valid = {};
for (const [index, [value, expected]] of cases.entries()) {
  if (module.Accepted(value) !== expected) throw new Error("javascript_composite.native_mismatch");
  valid[["none", "ok", "wrong_bytes", "error", "wrong_error"][index]] = expected;
}
for (const malformed of [{}, { tag: "some" }, { tag: "unknown" }]) {
  try { module.Accepted(malformed); } catch { continue; }
  throw new Error("javascript_composite.malformed_accepted");
}
console.log(JSON.stringify({ valid, malformed: 3 }));
