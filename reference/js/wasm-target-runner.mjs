import { readFile } from "node:fs/promises";

if (process.argv.length !== 6) {
  console.error("usage: node wasm-target-runner.mjs MODULE.wasm CURRENT DELTA LIMIT");
  process.exit(64);
}

const events = [];
const moduleBytes = await readFile(process.argv[2]);
const { instance } = await WebAssembly.instantiate(moduleBytes, {
  pulp: {
    log_bool(value) {
      events.push(value !== 0);
      return 0;
    },
  },
});
const requestPtr = instance.exports.pulp_alloc(24);
const namePtr = instance.exports.pulp_alloc(14);
const outPtr = instance.exports.pulp_alloc(8);
const memory = new DataView(instance.exports.memory.buffer);
memory.setBigInt64(requestPtr, BigInt(process.argv[3]), true);
memory.setBigInt64(requestPtr + 8, BigInt(process.argv[4]), true);
memory.setBigInt64(requestPtr + 16, BigInt(process.argv[5]), true);
new Uint8Array(instance.exports.memory.buffer, namePtr, 14).set(
  new TextEncoder().encode("quota.admit-v1"),
);
const status = instance.exports.pulp_on_call(namePtr, 14, requestPtr, 24, outPtr, outPtr + 4);
if (status !== 0) throw new Error(`pulp_on_call returned ${status}`);
const responsePtr = memory.getUint32(outPtr, true);
const responseLen = memory.getUint32(outPtr + 4, true);
if (responseLen !== 1) throw new Error(`invalid response length ${responseLen}`);
const result = memory.getUint8(responsePtr) !== 0;
process.stdout.write(`${JSON.stringify({ result, events })}\n`);
