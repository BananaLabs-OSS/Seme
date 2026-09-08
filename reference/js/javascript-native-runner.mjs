import assert from "node:assert/strict";

if (process.argv.length !== 5 && process.argv.length !== 6) throw new Error("usage: node javascript-native-runner.mjs MODULE LEFT RIGHT [FUNCTION]");
const module = await import(process.argv[2]);
const name = process.argv[5] || "Join";
assert.equal(typeof module[name], "function");
const result = module[name](process.argv[3], process.argv[4]);
assert.equal(typeof result, "string");
console.log(JSON.stringify({ result }));
