import assert from "node:assert/strict";
import fs from "node:fs";

if (process.argv.length !== 3) throw new Error("usage: node slice-abi-v2-runner.mjs FUNCTION.wasm");
const bytes = fs.readFileSync(process.argv[2]);

async function call(encoded) {
  const { instance } = await WebAssembly.instantiate(bytes, {});
  const api = instance.exports;
  const provider = new TextEncoder().encode("seme.function-v2");
  const request = Uint8Array.from(Buffer.from(encoded, "hex"));
  const providerPointer = api.pulp_alloc(provider.length);
  const requestPointer = request.length === 0 ? 0 : api.pulp_alloc(request.length);
  const outputPointer = api.pulp_alloc(8);
  const memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, providerPointer);
  memory.set(request, requestPointer);
  return api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
}

assert.equal(await call(""), 2, "truncated header must reject");
assert.equal(await call("0000000000000000"), 4, "noncanonical payload offset must reject");
assert.equal(await call("0800000001000000"), 4, "missing payload must reject");
assert.equal(await call("080000000000000000"), 4, "trailing payload must reject");
assert.equal(await call("0800000001020000"), 4, "513 elements must exceed the per-slice bound");
assert.equal(await call("0800000000020000" + "00".repeat(4096)), 0, "512 elements must remain valid");

assert.equal(await call("00".repeat(7161)), 2, "oversized requests must reject before decoding");
console.log(JSON.stringify({ malformed: "rejected", allocation_bound: 7160, maximum_elements: 512 }));
