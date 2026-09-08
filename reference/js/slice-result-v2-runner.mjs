import assert from "node:assert/strict";
import fs from "node:fs";

if (process.argv.length !== 3) throw new Error("usage: node slice-result-v2-runner.mjs FUNCTION.wasm");
const bytes = fs.readFileSync(process.argv[2]);

function encode(values, index, replacement, appended) {
  const result = Buffer.alloc(32 + values.length * 8);
  result.writeUInt32LE(32, 0);
  result.writeUInt32LE(values.length, 4);
  result.writeBigInt64LE(BigInt(index), 8);
  result.writeBigInt64LE(BigInt(replacement), 16);
  result.writeBigInt64LE(BigInt(appended), 24);
  values.forEach((value, position) => result.writeBigInt64LE(BigInt(value), 32 + position * 8));
  return result;
}

async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const provider = new TextEncoder().encode("seme.function-v2");
  const providerPointer = api.pulp_alloc(provider.length);
  const requestPointer = api.pulp_alloc(request.length);
  const outputPointer = api.pulp_alloc(8);
  let memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, providerPointer);
  memory.set(request, requestPointer);
  const before = Buffer.from(memory.subarray(requestPointer, requestPointer + request.length));
  const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
  const view = new DataView(api.memory.buffer);
  const responsePointer = view.getUint32(outputPointer, true);
  const responseLength = view.getUint32(outputPointer + 4, true);
  memory = new Uint8Array(api.memory.buffer);
  return { status, response: Buffer.from(memory.subarray(responsePointer, responsePointer + responseLength)), unchanged: before.equals(Buffer.from(memory.subarray(requestPointer, requestPointer + request.length))) };
}

const valid = await invoke(encode([-7, 0, 42], 1, 9, 100));
assert.equal(valid.status, 0);
assert.equal(valid.response.toString("hex"), "0800000004000000f9ffffffffffffff09000000000000002a000000000000006400000000000000");
assert.equal(valid.unchanged, true, "input payload must not alias the result update");
await assert.rejects(() => invoke(encode([-7, 0, 42], -1, 9, 100)), WebAssembly.RuntimeError);
await assert.rejects(() => invoke(encode([-7, 0, 42], 3, 9, 100)), WebAssembly.RuntimeError);
await assert.rejects(() => invoke(encode(new Array(512).fill(0), 0, 9, 100)), WebAssembly.RuntimeError);
console.log(JSON.stringify({ result: "exact", input: "unchanged", bounds: "trapped", append_limit: 512 }));
