import { readFile } from "node:fs/promises";

const path = process.argv[2];
if (!path) throw new Error("usage: node cumulative-text-collection-runner.mjs MODULE.wasm");
const bytes = await readFile(path);
const encoder = new TextEncoder();
const decoder = new TextDecoder("utf-8", { fatal: true });

async function run(prefix, values, want) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const text = encoder.encode(prefix);
  const sliceOffset = 16 + text.length;
  const size = sliceOffset + values.length * 8;
  const request = api.pulp_alloc(size);
  const output = api.pulp_alloc(4);
  const outputLength = api.pulp_alloc(4);
  const view = new DataView(api.memory.buffer);
  const memory = new Uint8Array(api.memory.buffer);

  view.setUint32(request, 16, true);
  view.setUint32(request + 4, text.length, true);
  view.setUint32(request + 8, sliceOffset, true);
  view.setUint32(request + 12, values.length, true);
  memory.set(text, request + 16);
  values.forEach((value, index) => view.setBigInt64(request + sliceOffset + index * 8, value, true));

  const before = memory.slice(request, request + size);
  if (api.pulp_on_call(0, 0, request, size, output, outputLength) !== 0) {
    throw new Error("call failed");
  }
  const response = view.getUint32(output, true);
  const length = view.getUint32(outputLength, true);
  const result = decoder.decode(memory.slice(response, response + length));
  if (result !== want) throw new Error(`bad result ${result}`);
  if (!before.every((value, index) => value === memory[request + index])) throw new Error("input mutated");
  return result;
}

async function rejectsMalformedInput(textBytes, textOffset, sliceOffset) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const size = 16 + textBytes.length;
  const request = api.pulp_alloc(size);
  const output = api.pulp_alloc(4);
  const outputLength = api.pulp_alloc(4);
  const view = new DataView(api.memory.buffer);
  const memory = new Uint8Array(api.memory.buffer);
  view.setUint32(request, textOffset, true);
  view.setUint32(request + 4, textBytes.length, true);
  view.setUint32(request + 8, sliceOffset, true);
  view.setUint32(request + 12, 0, true);
  memory.set(textBytes, request + 16);
  if (api.pulp_on_call(0, 0, request, size, output, outputLength) === 0) {
    throw new Error("malformed text request accepted");
  }
}

const positive = await run("sum", [3n, 4n], "sum:positive");
const negative = await run("sum", [-8n, 1n], "sum:non-positive");
const empty = await run("sum", [], "sum:non-positive");
await run("世界🚀", [1n], "世界🚀:positive");
await rejectsMalformedInput(new Uint8Array([0x61]), 17, 17);
await rejectsMalformedInput(new Uint8Array([0xc3, 0x28]), 16, 18);
console.log(JSON.stringify({ positive, negative, empty, unicode: "exact", composition: "existing-generic", input: "unchanged", malformed: "rejected" }));
