import fs from "node:fs";
import assert from "node:assert/strict";

const [, , wasmPath, requestHex, expectedHex, expectedEventsText, mode = "granted"] = process.argv;
const expectedEvents = expectedEventsText.split(",").filter(Boolean).map((value) => value === "true");
const events = [];
const imports = { pulp: { log_bool(value) { if (mode === "denied") return 99; events.push(value !== 0); return 0; } } };
const { instance } = await WebAssembly.instantiate(fs.readFileSync(wasmPath), imports);
const api = instance.exports;
const provider = new TextEncoder().encode("seme.function-v1");
const request = Uint8Array.from(Buffer.from(requestHex, "hex"));
const providerPointer = api.pulp_alloc(provider.length);
const requestPointer = api.pulp_alloc(request.length);
const outputPointer = api.pulp_alloc(8);
let memory = new Uint8Array(api.memory.buffer);
memory.set(provider, providerPointer);
memory.set(request, requestPointer);
if (mode === "denied") {
  assert.throws(() => api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4), WebAssembly.RuntimeError);
  assert.deepEqual(events, []);
  console.log(JSON.stringify({ mode, events }));
  process.exit(0);
}
const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
assert.equal(status, 0);
const view = new DataView(api.memory.buffer);
const responsePointer = view.getUint32(outputPointer, true);
const responseLength = view.getUint32(outputPointer + 4, true);
memory = new Uint8Array(api.memory.buffer);
const response = Buffer.from(memory.subarray(responsePointer, responsePointer + responseLength)).toString("hex");
assert.equal(response, expectedHex);
assert.deepEqual(events, expectedEvents);
console.log(JSON.stringify({ status, response, events }));
