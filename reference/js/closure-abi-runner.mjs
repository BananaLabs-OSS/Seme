import { readFile } from "node:fs/promises";

const [makePath, applyPath] = process.argv.slice(2);
if (!makePath || !applyPath) throw new Error("usage: node closure-abi-runner.mjs <make.wasm> <apply.wasm>");
const make = (await WebAssembly.instantiate(await readFile(makePath), {})).instance.exports;
const apply = (await WebAssembly.instantiate(await readFile(applyPath), {})).instance.exports;
const makeView = new DataView(make.memory.buffer);
const request = make.pulp_alloc(8), pointer = make.pulp_alloc(4), length = make.pulp_alloc(4);
makeView.setBigInt64(request, -7n, true);
if (make.pulp_on_call(0, 0, request, 8, pointer, length) !== 0) throw new Error("MakeAdder failed");
const closurePointer = makeView.getUint32(pointer, true);
if (makeView.getUint32(length, true) !== 16 || makeView.getBigUint64(closurePointer, true) !== 1n || makeView.getBigInt64(closurePointer + 8, true) !== -7n) throw new Error("invalid closure ABI");

const applyView = new DataView(apply.memory.buffer);
const applyRequest = apply.pulp_alloc(24), applyPointer = apply.pulp_alloc(4), applyLength = apply.pulp_alloc(4);
new Uint8Array(apply.memory.buffer, applyRequest, 16).set(new Uint8Array(make.memory.buffer, closurePointer, 16));
applyView.setBigInt64(applyRequest + 16, 12n, true);
if (apply.pulp_on_call(0, 0, applyRequest, 24, applyPointer, applyLength) !== 0) throw new Error("Apply failed");
const result = applyView.getBigInt64(applyView.getUint32(applyPointer, true), true);
if (result !== 5n) throw new Error(`unexpected closure result ${result}`);
applyView.setBigUint64(applyRequest, 2n, true);
let rejected = false;
try { apply.pulp_on_call(0, 0, applyRequest, 24, applyPointer, applyLength); }
catch (error) { rejected = error instanceof WebAssembly.RuntimeError; }
if (!rejected) throw new Error("unknown closure tag accepted");
console.log(JSON.stringify({ returnedClosure: "portable", result: result.toString(), unknownTag: "trapped" }));
