import fs from "node:fs";

if (process.argv.length !== 4) {
  throw new Error("usage: node pure-function-hardening.mjs MODULE.wasm VALID_REQUEST_HEX");
}
const moduleBytes = fs.readFileSync(process.argv[2]);

async function instantiate() {
  return (await WebAssembly.instantiate(moduleBytes, {})).instance.exports;
}

const allocator = await instantiate();
if (allocator.pulp_alloc(0) !== 0) throw new Error("zero allocation was not rejected");
const maximum = allocator.pulp_alloc(7160);
if (maximum !== 1024) throw new Error(`maximum arena allocation started at ${maximum}`);
if (allocator.pulp_alloc(1) !== 0) throw new Error("arena exhaustion was not rejected");
allocator.pulp_free(maximum, 7160);
if (allocator.pulp_alloc(7160) !== 1024) throw new Error("top allocation was not reclaimed");

const hostileFree = await instantiate();
const first = hostileFree.pulp_alloc(16);
if (first !== 1024) throw new Error(`hostile-free baseline started at ${first}`);
// Without an explicit arena upper bound, this sum wraps to the current 1048
// top and can forge a successful LIFO free: fffffff8 + 1048 + 8 == 1048 (u32).
hostileFree.pulp_free(0xfffffff8, 1048);
if (hostileFree.pulp_alloc(1) !== 1048) throw new Error("near-max forged free changed allocator state");

const foreignFree = await instantiate();
const older = foreignFree.pulp_alloc(16);
const newer = foreignFree.pulp_alloc(16);
if (older !== 1024 || newer !== 1048) throw new Error("foreign-free baseline mismatch");
foreignFree.pulp_free(older, 16);
foreignFree.pulp_free(8193, 1);
foreignFree.pulp_free(newer, 0);
if (foreignFree.pulp_alloc(1) !== 1072) throw new Error("foreign or malformed free changed allocator state");

const api = await instantiate();
const provider = new TextEncoder().encode("seme.function-v1");
const request = Buffer.from(process.argv[3], "hex");
for (let index = 0; index < 20000; index++) {
  const providerPointer = api.pulp_alloc(provider.length);
  const requestPointer = api.pulp_alloc(request.length);
  const outputPointer = api.pulp_alloc(8);
  if (!providerPointer || !requestPointer || !outputPointer) throw new Error(`allocation failed at call ${index}`);
  const memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, providerPointer);
  memory.set(request, requestPointer);
  const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
  if (status !== 0) throw new Error(`call ${index} returned ${status}`);
  const view = new DataView(api.memory.buffer);
  const responsePointer = view.getUint32(outputPointer, true);
  const responseLength = view.getUint32(outputPointer + 4, true);
  api.pulp_free(responsePointer, responseLength);
  api.pulp_free(outputPointer, 8);
  api.pulp_free(requestPointer, request.length);
  api.pulp_free(providerPointer, provider.length);
}
if (api.pulp_alloc(7160) !== 1024) throw new Error("repeated calls leaked arena allocations");
console.log("pure function allocator: bounds, forged-free rejection, and 20000-call lifetime passed");
