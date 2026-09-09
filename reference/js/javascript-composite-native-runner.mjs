import { Seme } from "./seme-values.mjs";

globalThis.Seme = Seme;
const module = await import(new URL("../../fixtures/javascript-uab-02/composite.js", import.meta.url));
const cases = [
  [Seme.none(), false],
  [Seme.some(Seme.ok(Seme.bytes([111, 107]))), true],
  [Seme.some(Seme.ok(Seme.bytes([110, 111]))), false],
  [Seme.some(Seme.error("bad")), true],
  [Seme.some(Seme.error("no")), false],
];
for (const [value, expected] of cases) {
  if (module.Accepted(value) !== expected) throw new Error("javascript_composite.native_mismatch");
}
for (const malformed of [{}, { tag: "some" }, { tag: "unknown" }]) {
  try { module.Accepted(malformed); } catch { continue; }
  throw new Error("javascript_composite.malformed_accepted");
}
console.log(JSON.stringify({ native: "javascript", vectors: cases.length, malformed: 3 }));
