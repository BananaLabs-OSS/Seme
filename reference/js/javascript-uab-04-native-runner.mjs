import fs from "node:fs";
import { pathToFileURL } from "node:url";
import { Seme } from "./seme-values.mjs";

const [sourcePath, vectorsPath] = process.argv.slice(2);
if (!sourcePath || !vectorsPath) throw new Error("usage: javascript-uab-04-native-runner SOURCE VECTORS");
globalThis.Seme = Seme;
const program = await import(`${pathToFileURL(sourcePath)}?uab04=${Date.now()}`);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const invoke = item => program.Evaluate(item.values.map(BigInt), BigInt(item.index), BigInt(item.replacement), BigInt(item.appended), BigInt(item.removeIndex), BigInt(item.keepKey), BigInt(item.removeKey));
const valid = {};
for (const item of vectors.valid) {
  const original = item.values.slice();
  valid[item.name] = invoke(item).toString();
  if (valid[item.name] !== item.result) throw new Error(`javascript.uab04.mismatch:${item.name}`);
  if (JSON.stringify(item.values) !== JSON.stringify(original)) throw new Error(`javascript.uab04.mutated_input:${item.name}`);
}
let malformed = 0;
for (const item of vectors.malformed) {
  try {
    if (item.category === "collection-kind") program.Evaluate("not-a-slice", BigInt(item.index), BigInt(item.replacement), BigInt(item.appended), BigInt(item.removeIndex), BigInt(item.keepKey), BigInt(item.removeKey));
    else invoke(item);
  } catch { malformed += 1; }
}
if (malformed !== vectors.malformed.length) throw new Error("javascript.uab04.malformed_accepted");
console.log(JSON.stringify({valid, malformed}));
