import fs from "node:fs";

if (process.argv.length < 4) {
  throw new Error("usage: node pure-function-runner.mjs MODULE.wasm REQUEST_HEX...");
}
const bytes = fs.readFileSync(process.argv[2]);
const { instance } = await WebAssembly.instantiate(bytes, {});
const api = instance.exports;
if (api.pulp_init(0, 0) !== 0) throw new Error("pulp_init failed");
const encoder = new TextEncoder();
const provider = encoder.encode("seme.function-v1");
for (const encoded of process.argv.slice(3)) {
  const request = Buffer.from(encoded, "hex");
  const providerPointer = api.pulp_alloc(provider.length);
  const requestPointer = api.pulp_alloc(request.length);
  const outputPointer = api.pulp_alloc(8);
  const memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, providerPointer);
  memory.set(request, requestPointer);
  const status = api.pulp_on_call(providerPointer, provider.length, requestPointer, request.length, outputPointer, outputPointer + 4);
  const view = new DataView(api.memory.buffer);
  const responsePointer = view.getUint32(outputPointer, true);
  const responseLength = view.getUint32(outputPointer + 4, true);
  const response = Buffer.from(new Uint8Array(api.memory.buffer, responsePointer, responseLength)).toString("hex");
  console.log(JSON.stringify({ request: encoded, status, response }));
  if (responseLength !== 0) api.pulp_free(responsePointer, responseLength);
  api.pulp_free(outputPointer, 8);
  if (request.length !== 0) api.pulp_free(requestPointer, request.length);
  api.pulp_free(providerPointer, provider.length);
}
if (api.pulp_shutdown() !== 0) throw new Error("pulp_shutdown failed");
