import fs from "node:fs";
import { pathToFileURL } from "node:url";
import { Seme } from "./seme-values.mjs";
globalThis.Seme=Seme;
const[sourcePath,vectorsPath]=process.argv.slice(2),program=await import(`${pathToFileURL(sourcePath)}?uab08=${Date.now()}`),vectors=JSON.parse(fs.readFileSync(vectorsPath,"utf8")),valid={};
for(const item of vectors.valid){const observed=program.IncrementPositive(BigInt(item.value)),actual={variant:observed.tag,payload:typeof observed.value==="bigint"?observed.value.toString():observed.value};if(actual.variant!==item.variant||actual.payload!==item.payload)throw Error(`javascript.uab08.mismatch:${item.name}`);valid[item.name]=actual}
console.log(JSON.stringify({valid}));
