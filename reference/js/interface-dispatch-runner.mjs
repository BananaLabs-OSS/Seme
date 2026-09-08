import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node interface-dispatch-runner.mjs <module.wasm>");
const { instance } = await WebAssembly.instantiate(await readFile(wasmPath), {});
const view = new DataView(instance.exports.memory.buffer);

function invoke(useScale, amount, value, expected) {
  const request = instance.exports.pulp_alloc(17);
  const responsePointer = instance.exports.pulp_alloc(4);
  const responseLength = instance.exports.pulp_alloc(4);
  view.setUint8(request, useScale);
  view.setBigInt64(request + 1, amount, true);
  view.setBigInt64(request + 9, value, true);
  const status = instance.exports.pulp_on_call(0, 0, request, 17, responsePointer, responseLength);
  if (status !== 0) throw new Error(`provider status ${status}`);
  const output = view.getUint32(responsePointer, true);
  const length = view.getUint32(responseLength, true);
  const result = view.getBigInt64(output, true);
  if (length !== 8 || result !== expected) throw new Error(`unexpected result ${result}, length ${length}`);
  if (view.getUint8(request) !== useScale || view.getBigInt64(request + 1, true) !== amount || view.getBigInt64(request + 9, true) !== value) throw new Error("provider mutated request");
  return result;
}

const offset = invoke(0, 3n, 4n, 7n);
const scale = invoke(1, 3n, 4n, 12n);
invoke(1, -1n, 9223372036854775807n, -9223372036854775807n);
const malformed = instance.exports.pulp_alloc(17);
const malformedPointer = instance.exports.pulp_alloc(4);
const malformedLength = instance.exports.pulp_alloc(4);
if (instance.exports.pulp_on_call(0, 0, malformed, 16, malformedPointer, malformedLength) === 0) throw new Error("short request accepted");
if (instance.exports.pulp_on_call(0, 0, malformed, 18, malformedPointer, malformedLength) === 0) throw new Error("long request accepted");
view.setUint8(malformed, 2);
let rejectedBoolean = false;
try { instance.exports.pulp_on_call(0, 0, malformed, 17, malformedPointer, malformedLength); }
catch (error) { rejectedBoolean = error instanceof WebAssembly.RuntimeError; }
if (!rejectedBoolean) throw new Error("non-canonical Boolean accepted");
console.log(JSON.stringify({ offset: offset.toString(), scale: scale.toString(), request: "unchanged", overflow: "wrapped", malformed: "rejected", boolean: "canonical" }));
