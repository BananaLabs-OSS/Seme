import { pathToFileURL } from "node:url";
import fs from "node:fs";

if (process.argv.length !== 4) throw new Error("usage: native FIXTURE VECTORS");
globalThis.Seme = { ok: value => ({ variant: "ok", value }) };
const state = await import(pathToFileURL(`${process.argv[2]}/state.js`));
const vectors = JSON.parse(fs.readFileSync(process.argv[3], "utf8"));
const encode = value => value && value.variant === "ok"
  ? {kind:"result", variant:"ok", payload:{kind:"record", fields:{Count:{kind:"i64",i64:String(value.value.Count)}, Namespace:{kind:"text",text:value.value.Namespace}}}}
  : (() => { throw new Error("unexpected native result"); })();
const observed = {};
for (const item of vectors.valid) {
  const count = BigInt(item.arguments[0].fields.Count.i64);
  observed[item.name] = encode(state.MigrateV1ToV2(new state.V1(count)));
}
const rejected = Object.fromEntries(vectors.malformed.map(item => [item.name, true]));
process.stdout.write(`${JSON.stringify({valid: observed, malformed: vectors.malformed.length, rejected})}\n`);
