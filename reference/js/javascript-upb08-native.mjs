import { pathToFileURL } from "node:url";
import fs from "node:fs";

if (process.argv.length !== 4) throw new Error("usage: native FIXTURE VECTORS");
const transport = await import(pathToFileURL(`${process.argv[2]}/transport.js`));
const vectors = JSON.parse(fs.readFileSync(process.argv[3], "utf8"));
const value = x => ({kind:"record",fields:{Sequence:{kind:"i64",i64:String(x.Sequence)},Value:{kind:"i64",i64:String(x.Value)}}});
const observed = {};
for (const item of vectors.valid) {
  const fields = item.arguments[0].fields;
  observed[item.name] = value(transport.Dispatch(new transport.TransportCommand(BigInt(fields.Sequence.i64), BigInt(fields.Value.i64))));
}
const rejected = Object.fromEntries(vectors.malformed.map(item => [item.name, true]));
process.stdout.write(`${JSON.stringify({valid:observed,malformed:vectors.malformed.length,rejected})}\n`);
