import fs from "node:fs";

const [wasmPath, ...arguments_] = process.argv.slice(2);
if (!wasmPath) throw new Error("usage: pure-vm-scalar-runner WASM [I64 ...]");
const instance = await WebAssembly.instantiate(fs.readFileSync(wasmPath), {});
const result = instance.instance.exports.run(...arguments_.map(BigInt));
process.stdout.write(`${result}\n`);
