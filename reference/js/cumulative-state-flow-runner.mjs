import { readFile } from "node:fs/promises";

const path = process.argv[2];
if (!path) throw new Error("usage: node cumulative-state-flow-runner.mjs MODULE.wasm");
const bytes = await readFile(path);

async function run(state, values, enabled, want) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const size = 17 + 8 * values.length;
  const request = api.pulp_alloc(size);
  const output = api.pulp_alloc(4);
  const outputLength = api.pulp_alloc(4);
  const view = new DataView(api.memory.buffer);

  view.setBigInt64(request, state, true);
  view.setUint32(request + 8, 17, true);
  view.setUint32(request + 12, values.length, true);
  view.setUint8(request + 16, enabled ? 1 : 0);
  values.forEach((value, index) => view.setBigInt64(request + 17 + index * 8, value, true));

  const before = new Uint8Array(api.memory.buffer, request, size).slice();
  if (api.pulp_on_call(0, 0, request, size, output, outputLength) !== 0) {
    throw new Error("call failed");
  }
  const response = view.getUint32(output, true);
  const nextState = view.getBigInt64(response, true);
  const result = view.getBigInt64(response + 8, true);
  if (view.getUint32(outputLength, true) !== 16 || nextState !== want || result !== want) {
    throw new Error(`bad transition ${nextState}/${result}`);
  }
  const after = new Uint8Array(api.memory.buffer, request, size);
  if (!before.every((value, index) => value === after[index])) throw new Error("input mutated");
  return result;
}

async function rejectsMalformedDescriptor() {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const request = api.pulp_alloc(17);
  const output = api.pulp_alloc(4);
  const outputLength = api.pulp_alloc(4);
  const view = new DataView(api.memory.buffer);
  view.setUint32(request + 8, 18, true);
  view.setUint32(request + 12, 1, true);
  if (api.pulp_on_call(0, 0, request, 17, output, outputLength) === 0) {
    throw new Error("malformed slice descriptor accepted");
  }
}

const enabled = await run(-7n, [3n, 4n, 5n], true, 5n);
const disabled = await run(-7n, [3n, 4n, 5n], false, -7n);
await run(9223372036854775807n, [1n], true, -9223372036854775808n);
await run(4n, [], true, 4n);
await rejectsMalformedDescriptor();
console.log(JSON.stringify({ enabled: String(enabled), disabled: String(disabled), composition: "generic", input: "unchanged", overflow: "wrapped", malformed: "rejected" }));
