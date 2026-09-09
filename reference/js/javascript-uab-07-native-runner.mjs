import fs from "node:fs";
import { pathToFileURL } from "node:url";
const [sourcePath,vectorsPath]=process.argv.slice(2),program=await import(`${pathToFileURL(sourcePath)}?uab07=${Date.now()}`),vectors=JSON.parse(fs.readFileSync(vectorsPath,"utf8")),valid={};
for(const item of vectors.valid){const counter={Value:BigInt(item.value)},observed=program.Step(counter,BigInt(item.delta));if(counter.Value.toString()!==item.value)throw Error(`javascript.uab07.mutated:${item.name}`);const actual={state:observed.state.Value.toString(),result:observed.result.toString()};if(actual.state!==item.state||actual.result!==item.result)throw Error(`javascript.uab07.mismatch:${item.name}`);valid[item.name]=actual}
console.log(JSON.stringify({valid}));
