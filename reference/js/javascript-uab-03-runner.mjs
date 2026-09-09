import fs from "node:fs";
import { pathToFileURL } from "node:url";
import { Seme } from "./seme-values.mjs";
globalThis.Seme = Seme;

const [programPath, vectorsPath] = process.argv.slice(2);
if (!programPath || !vectorsPath) throw new Error("usage: javascript-uab-03-runner PROGRAM VECTORS");
const module = await import(programPath.startsWith("file:") ? programPath : pathToFileURL(programPath));
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const i64 = (value) => {
  if (typeof value !== "string" || !/^-?(?:0|[1-9][0-9]*)$/.test(value)) throw new TypeError("javascript.i64_boundary");
  const parsed = BigInt(value);
  if (parsed < -(1n << 63n) || parsed > (1n << 63n) - 1n) throw new RangeError("javascript.i64_boundary");
  return parsed;
};
const boolean = (value) => {
  if (typeof value !== "boolean") throw new TypeError("javascript.boolean_boundary");
  return value;
};
const valid = {};
for (const item of vectors.valid) {
  const observed = module.Evaluate(i64(item.arguments[0]), i64(item.arguments[1]), boolean(item.arguments[2]), i64(item.arguments[3]), i64(item.arguments[4]));
  if (typeof observed !== "bigint" || observed.toString() !== item.result) throw new Error(`javascript.uab03.mismatch:${item.name}`);
  valid[item.name] = observed.toString();
}
let malformed = 0;
for (const args of [["9","4",true,"1","0"],["9","4",false,"0","1"],[9,"4",true,"0","0"],["9","4",1,"0","0"]]) {
  try { module.Evaluate(i64(args[0]),i64(args[1]),boolean(args[2]),i64(args[3]),i64(args[4])); } catch { malformed += 1; }
}
if (malformed !== 4) throw new Error("javascript.uab03.malformed_accepted");
console.log(JSON.stringify({ valid, malformed }));
