import assert from "node:assert/strict";

if (process.argv.length !== 5) throw new Error("usage: node javascript-native-runner.mjs MODULE LEFT RIGHT");
const module = await import(process.argv[2]);
assert.equal(typeof module.Join, "function");
const result = module.Join(process.argv[3], process.argv[4]);
assert.equal(typeof result, "string");
console.log(JSON.stringify({ result }));
