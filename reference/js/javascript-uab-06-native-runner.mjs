import fs from "node:fs";
import { pathToFileURL } from "node:url";

const [sourcePath, vectorsPath, selected] = process.argv.slice(2);
const program = await import(`${pathToFileURL(sourcePath)}?uab06=${Date.now()}`);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const valid = {};
for (const family of selected ? [selected] : ["immutable", "mutable"]) {
  const fn = family === "immutable" ? program.Immutable : program.Mutable;
  for (const item of vectors[family]) {
    const observed = fn(...item.arguments.map(BigInt)).toString();
    if (observed !== item.result) throw new Error(`javascript.uab06.mismatch:${item.name}`);
    valid[item.name] = observed;
  }
}
console.log(JSON.stringify({ valid }));
