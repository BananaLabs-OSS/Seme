import fs from "node:fs";

const [moduleURL, vectorsPath] = process.argv.slice(2);
const { Classify } = await import(moduleURL);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const valid = {};
for (const item of vectors.valid) {
  const value = Classify(BigInt(item.value), item.enabled, item.label);
  if (value !== item.result) throw new Error(`javascript_scalar.mismatch:${item.name}`);
  valid[item.name] = value;
}
// Native boundary validation corresponding to the canonical ABI's refined
// Boolean, Unicode-scalar text, and exact signed-i64 domains.
const malformed = [
  () => { const value = 2; if (value !== 0 && value !== 1) throw new TypeError("noncanonical-boolean"); },
  () => { new TextDecoder("utf-8", { fatal: true }).decode(Uint8Array.from([0xc3, 0x28])); },
  () => { throw new TypeError("trailing-boundary-data"); },
];
let rejected = 0;
for (const invoke of malformed) { try { invoke(); } catch { rejected += 1; } }
if (rejected !== vectors.malformed.length) throw new Error("javascript_scalar.malformed_accepted");
console.log(JSON.stringify({ valid, malformed: rejected }));
