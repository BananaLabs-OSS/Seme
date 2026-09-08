import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node immutable-closure-runner.mjs <module.wasm>");
const { instance } = await WebAssembly.instantiate(await readFile(wasmPath), {});
const view = new DataView(instance.exports.memory.buffer);

function invoke(base, value, expected) {
  const request = instance.exports.pulp_alloc(16);
  const responsePointer = instance.exports.pulp_alloc(4);
  const responseLength = instance.exports.pulp_alloc(4);
  view.setBigInt64(request, base, true);
  view.setBigInt64(request + 8, value, true);
  const status = instance.exports.pulp_on_call(0, 0, request, 16, responsePointer, responseLength);
  if (status !== 0) throw new Error(`provider status ${status}`);
  const output = view.getUint32(responsePointer, true);
  const length = view.getUint32(responseLength, true);
  const result = view.getBigInt64(output, true);
  if (length !== 8 || result !== expected) throw new Error(`unexpected result ${result}, length ${length}`);
  if (view.getBigInt64(request, true) !== base || view.getBigInt64(request + 8, true) !== value) throw new Error("capture source mutated");
  return result;
}

const positive = invoke(7n, 5n, 12n);
const negative = invoke(-7n, 5n, -2n);
invoke(9223372036854775807n, 1n, -9223372036854775808n);
const malformed = instance.exports.pulp_alloc(16);
const pointer = instance.exports.pulp_alloc(4);
const length = instance.exports.pulp_alloc(4);
if (instance.exports.pulp_on_call(0, 0, malformed, 15, pointer, length) === 0) throw new Error("short request accepted");
if (instance.exports.pulp_on_call(0, 0, malformed, 17, pointer, length) === 0) throw new Error("long request accepted");
console.log(JSON.stringify({ positive: positive.toString(), negative: negative.toString(), capture: "immutable", overflow: "wrapped", malformed: "rejected" }));
