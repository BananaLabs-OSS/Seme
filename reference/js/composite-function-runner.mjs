import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

const path = process.argv[2];
if (!path) throw new Error("usage: node composite-function-runner.mjs MODULE.wasm");
const wasm = await readFile(path);

const none = Buffer.alloc(10);
const some = (tag, payload) => {
  const request = Buffer.alloc(10 + payload.length);
  request[0] = 1;
  request[1] = tag;
  request.writeUInt32LE(10, 2);
  request.writeUInt32LE(payload.length, 6);
  payload.copy(request, 10);
  return request;
};

async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(wasm, {});
  const api = instance.exports;
  const input = api.pulp_alloc(request.length);
  const output = api.pulp_alloc(4);
  const outputLength = api.pulp_alloc(4);
  const memory = new Uint8Array(api.memory.buffer);
  memory.set(request, input);
  const before = memory.slice(input, input + request.length);
  const status = api.pulp_on_call(0, 0, input, request.length, output, outputLength);
  assert.deepEqual(memory.slice(input, input + request.length), before, "input mutated");
  if (status !== 0) return { status };
  const view = new DataView(api.memory.buffer);
  const response = view.getUint32(output, true);
  assert.equal(view.getUint32(outputLength, true), 1, "noncanonical response length");
  const value = memory[response];
  assert.ok(value === 0 || value === 1, "noncanonical Boolean response");
  return { status, value };
}

assert.deepEqual(await invoke(none), { status: 0, value: 0 });
assert.deepEqual(await invoke(some(0, Buffer.from("ok"))), { status: 0, value: 1 });
assert.deepEqual(await invoke(some(0, Buffer.from("no"))), { status: 0, value: 0 });
assert.deepEqual(await invoke(some(1, Buffer.from("bad"))), { status: 0, value: 1 });
assert.deepEqual(await invoke(some(1, Buffer.from("no"))), { status: 0, value: 0 });

const malformed = [];
malformed.push(Buffer.from([2, 0, 0, 0, 0, 0, 0, 0, 0, 0])); // option tag
malformed.push(Buffer.from([1, 2, 10, 0, 0, 0, 0, 0, 0, 0])); // result tag
malformed.push(Buffer.from([0, 1, 0, 0, 0, 0, 0, 0, 0, 0])); // inactive bytes
malformed.push(Buffer.concat([none, Buffer.from([0])])); // absent payload
malformed.push(Buffer.from([1, 0, 9, 0, 0, 0, 1, 0, 0, 0, 0])); // descriptor before fixed area
malformed.push(Buffer.from([1, 0, 10, 0, 0, 0, 2, 0, 0, 0, 0])); // descriptor outside request
malformed.push(Buffer.from([1, 0, 11, 0, 0, 0, 1, 0, 0, 0, 0])); // noncontiguous payload
malformed.push(some(1, Buffer.from([0xc3, 0x28]))); // invalid UTF-8 text
for (const request of malformed) assert.notEqual((await invoke(request)).status, 0, "malformed request accepted");

console.log("composite Wasm: valid variants executed and malformed encodings rejected");
