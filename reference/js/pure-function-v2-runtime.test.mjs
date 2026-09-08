import fs from "node:fs";
import assert from "node:assert/strict";
import { packTextFields } from "./string-abi-v2.mjs";

if (process.argv.length !== 4) throw new Error("usage: node pure-function-v2-runtime.test.mjs JOIN.wasm EQUAL.wasm");

async function load(path) {
  const { instance } = await WebAssembly.instantiate(fs.readFileSync(path), {});
  return instance.exports;
}

function call(api, request) {
  const provider = new TextEncoder().encode("seme.function-v2");
  const providerPointer = api.pulp_alloc(provider.length);
  const requestPointer = api.pulp_alloc(request.length);
  const outputPointer = api.pulp_alloc(8);
  const memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, providerPointer);
  memory.set(request, requestPointer);
  memory.fill(0, outputPointer, outputPointer + 8);
  const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
  const view = new DataView(api.memory.buffer);
  const pointer = view.getUint32(outputPointer, true);
  const length = view.getUint32(outputPointer + 4, true);
  const response = status === 0 ? Uint8Array.from(new Uint8Array(api.memory.buffer, pointer, length)) : new Uint8Array();
  api.pulp_free(outputPointer, 8);
  if (request.length !== 0) api.pulp_free(requestPointer, request.length);
  api.pulp_free(providerPointer, provider.length);
  return { status, response };
}

const join = await load(process.argv[2]);
const equal = await load(process.argv[3]);
const decoder = new TextDecoder("utf-8", { fatal: true });

let result = call(join, packTextFields(["雪", "🦀"]));
assert.equal(result.status, 0);
assert.equal(decoder.decode(result.response), "雪🦀");
result = call(join, packTextFields(["", ""]));
assert.equal(result.status, 0);
assert.equal(result.response.length, 0);
result = call(join, packTextFields(["a".repeat(2048), "b".repeat(2048)]));
assert.equal(result.status, 0);
assert.equal(result.response.length, 4096);
result = call(join, packTextFields(["a".repeat(4096), "b"]));
assert.equal(result.status, 6);
for (let index = 0; index < 2000; index += 1) {
  result = call(join, packTextFields(["雪", "🦀"]));
  assert.equal(result.status, 0);
}

result = call(equal, packTextFields(["same", "same"]));
assert.deepEqual([...result.response], [1]);
result = call(equal, packTextFields(["same", "different"]));
assert.deepEqual([...result.response], [0]);

const malformed = Uint8Array.from(packTextFields(["aa", ""]));
malformed[16] = 0xc0; malformed[17] = 0x80;
assert.equal(call(equal, malformed).status, 5);
const gap = Uint8Array.from(packTextFields(["a", "b"]));
new DataView(gap.buffer).setUint32(0, 17, true);
assert.equal(call(equal, gap).status, 4);
assert.equal(call(equal, new Uint8Array(15)).status, 2);

console.log("pure function ABI v2 runtime vectors: ok");
