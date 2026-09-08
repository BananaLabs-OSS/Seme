import fs from "node:fs";
import assert from "node:assert/strict";

if (process.argv.length !== 5) {
  throw new Error("usage: node pure-function-v2-runner.mjs FUNCTION.wasm REQUEST_HEX EXPECTED_RESPONSE_HEX");
}
const [, , wasmPath, requestHex, expectedHex] = process.argv;
for (const value of [requestHex, expectedHex]) {
  if (!/^(?:[0-9a-fA-F]{2})*$/.test(value)) throw new Error("pure_function_v2.invalid_hex");
}

const { instance } = await WebAssembly.instantiate(fs.readFileSync(wasmPath), {});
const api = instance.exports;
const provider = new TextEncoder().encode("seme.function-v2");
const request = Uint8Array.from(Buffer.from(requestHex, "hex"));
const providerPointer = api.pulp_alloc(provider.length);
const requestPointer = api.pulp_alloc(request.length);
const outputPointer = api.pulp_alloc(8);
let memory = new Uint8Array(api.memory.buffer);
memory.set(provider, providerPointer);
memory.set(request, requestPointer);
memory.fill(0, outputPointer, outputPointer + 8);
const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
assert.equal(status, 0);
const view = new DataView(api.memory.buffer);
const responsePointer = view.getUint32(outputPointer, true);
const responseLength = view.getUint32(outputPointer + 4, true);
memory = new Uint8Array(api.memory.buffer);
const response = Buffer.from(memory.subarray(responsePointer, responsePointer + responseLength)).toString("hex");
assert.equal(response, expectedHex.toLowerCase());
console.log(JSON.stringify({ status, response }));
