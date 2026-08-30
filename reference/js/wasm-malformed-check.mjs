import { readFile } from "node:fs/promises";

if (process.argv.length !== 3) {
  console.error("usage: node wasm-malformed-check.mjs MODULE.wasm");
  process.exit(64);
}

const events = [];
const { instance } = await WebAssembly.instantiate(await readFile(process.argv[2]), {
  pulp: { log_bool(value) { events.push(value); return 0; } },
});
const requestPtr = instance.exports.pulp_alloc(31);
const namePtr = instance.exports.pulp_alloc(14);
const outPtr = instance.exports.pulp_alloc(8);
new Uint8Array(instance.exports.memory.buffer, namePtr, 14).set(new TextEncoder().encode("quota.admit-v1"));
const status = instance.exports.pulp_on_call(namePtr, 14, requestPtr, 31, outPtr, outPtr + 4);
if (status !== 2) throw new Error(`malformed request status ${status}, want 2`);
if (events.length !== 0) throw new Error(`malformed request performed ${events.length} effects`);
