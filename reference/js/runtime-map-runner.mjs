import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node runtime-map-runner.mjs <module.wasm>");
const bytes = await readFile(wasmPath);

async function invoke(values, key, expected) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const size = 16 + values.length * 8;
  const request = api.pulp_alloc(size), pointer = api.pulp_alloc(4), length = api.pulp_alloc(4);
  const view = new DataView(api.memory.buffer);
  view.setUint32(request, 16, true);
  view.setUint32(request + 4, values.length, true);
  view.setBigInt64(request + 8, key, true);
  values.forEach((value, index) => view.setBigInt64(request + 16 + index * 8, value, true));
  const before = new Uint8Array(api.memory.buffer, request, size).slice();
  const status = api.pulp_on_call(0, 0, request, size, pointer, length);
  if (status !== 0) throw new Error(`provider status ${status}`);
  const result = view.getBigInt64(view.getUint32(pointer, true), true);
  if (view.getUint32(length, true) !== 8 || result !== expected) throw new Error(`unexpected tally ${result}`);
  if (!before.every((value, index) => value === new Uint8Array(api.memory.buffer, request, size)[index])) throw new Error("map realization aliased request");
  return result;
}

const repeated = await invoke([3n, -2n, 3n, 7n, -2n, 3n], 3n, 3n);
const negative = await invoke([3n, -2n, 3n, 7n, -2n, 3n], -2n, 2n);
const missing = await invoke([3n, -2n, 3n], 99n, 0n);
const reordered = await invoke([-2n, 3n, 3n, -2n, 7n, 3n], 3n, 3n);
await invoke([], 1n, 0n);
await invoke(Array(512).fill(-9n), -9n, 512n);

async function malformed(encoded) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports, request = api.pulp_alloc(encoded.length), pointer = api.pulp_alloc(4), length = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(encoded, request);
  return api.pulp_on_call(0, 0, request, encoded.length, pointer, length);
}
if (await malformed(new Uint8Array(15)) === 0) throw new Error("truncated header accepted");
const tooMany = new Uint8Array(16 + 513 * 8); new DataView(tooMany.buffer).setUint32(0, 16, true); new DataView(tooMany.buffer).setUint32(4, 513, true);
if (await malformed(tooMany) === 0) throw new Error("513 elements accepted");
const badOffset = new Uint8Array(16); new DataView(badOffset.buffer).setUint32(0, 8, true);
if (await malformed(badOffset) === 0) throw new Error("noncanonical payload offset accepted");
console.log(JSON.stringify({ repeated:repeated.toString(),negative:negative.toString(),missing:missing.toString(),reordered:reordered.toString(),order:"independent",maximumElements:512,malformed:"rejected",alias:"none" }));
