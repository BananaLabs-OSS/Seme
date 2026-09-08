import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node state-transition-runner.mjs <module.wasm>");

const { instance } = await WebAssembly.instantiate(await readFile(wasmPath), {});
const view = new DataView(instance.exports.memory.buffer);
function invoke(receiver, delta, expected) {
  const request = instance.exports.pulp_alloc(16);
  const responsePointer = instance.exports.pulp_alloc(4);
  const responseLength = instance.exports.pulp_alloc(4);
  if (!request || !responsePointer || !responseLength) throw new Error("allocation failed");
  view.setBigInt64(request, receiver, true);
  view.setBigInt64(request + 8, delta, true);
  const status = instance.exports.pulp_on_call(0, 0, request, 16, responsePointer, responseLength);
  if (status !== 0) throw new Error(`provider status ${status}`);
  const output = view.getUint32(responsePointer, true);
  const length = view.getUint32(responseLength, true);
  const state = view.getBigInt64(output, true);
  const result = view.getBigInt64(output + 8, true);
  if (length !== 16 || state !== expected || result !== expected) throw new Error(`unexpected transition ${state}, ${result}, length ${length}`);
  if (view.getBigInt64(request, true) !== receiver || view.getBigInt64(request + 8, true) !== delta) throw new Error("provider mutated its value receiver request");
  return { state, result };
}

const ordinary = invoke(-7n, 12n, 5n);
invoke(9223372036854775807n, 1n, -9223372036854775808n);
const invalid = instance.exports.pulp_alloc(16);
const invalidPointer = instance.exports.pulp_alloc(4);
const invalidLength = instance.exports.pulp_alloc(4);
if (instance.exports.pulp_on_call(0, 0, invalid, 15, invalidPointer, invalidLength) === 0) throw new Error("short request accepted");
if (instance.exports.pulp_on_call(0, 0, invalid, 17, invalidPointer, invalidLength) === 0) throw new Error("long request accepted");
console.log(JSON.stringify({ state: ordinary.state.toString(), result: ordinary.result.toString(), receiver: "unchanged", overflow: "wrapped", malformed: "rejected" }));
