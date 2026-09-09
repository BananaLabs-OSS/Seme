import fs from "node:fs";
import { Seme } from "./seme-values.mjs";

const [moduleURL, vectorsPath] = process.argv.slice(2);
if (!moduleURL || !vectorsPath) throw new Error("usage: javascript-aggregate-native-runner MODULE VECTORS.json");
globalThis.Seme = Seme;
const { Observe } = await import(moduleURL);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const valid = {};
for (const item of vectors.valid) {
  const result = Observe({ value: BigInt(item.record) }, Seme.array(item.array.map(BigInt)), item.slice.map(BigInt), new Map(item.map.map(([key, value]) => [BigInt(key), BigInt(value)])), BigInt(item.key));
  if (result !== BigInt(item.result)) throw new Error(`javascript_aggregate.mismatch:${item.name}`);
  valid[item.name] = result.toString();
}
const invalid = [
  () => Observe({}, Seme.array([1n, 2n]), [], new Map(), 0n),
  () => Observe({ value: 0n }, Seme.array([1n]), [], new Map(), 0n),
  () => Observe({ value: 0n }, Seme.array([1n, 2n]), [], {}, 0n),
];
let malformed = 0;
for (const invoke of invalid) { try { invoke(); } catch { malformed += 1; } }
if (malformed !== vectors.malformed.length) throw new Error("javascript_aggregate.malformed_accepted");
console.log(JSON.stringify({ valid, malformed }));
