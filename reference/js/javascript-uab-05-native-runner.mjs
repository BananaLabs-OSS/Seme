import fs from "node:fs";
import { pathToFileURL } from "node:url";
const [sourcePath, vectorsPath] = process.argv.slice(2);
const program = await import(`${pathToFileURL(sourcePath)}?uab05=${Date.now()}`);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const valid = {};
for (const item of vectors.valid) {
  const result = program.Dispatch(item.scale, BigInt(item.amount), BigInt(item.value)).toString();
  if (result !== item.result) throw new Error(`javascript.uab05.mismatch:${item.name}`);
  valid[item.name] = result;
}
console.log(JSON.stringify({valid}));
