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
const subject = new TextEncoder().encode("tenant-a");
const evidence = Uint8Array.of(1, 2, 3);
const requestLength = 32 + subject.length + evidence.length;
const requestPtr = instance.exports.pulp_alloc(requestLength);
const namePtr = instance.exports.pulp_alloc(14);
const outPtr = instance.exports.pulp_alloc(8);
const memory = new DataView(instance.exports.memory.buffer);
memory.setBigInt64(requestPtr, BigInt(process.argv[3]), true);
memory.setBigInt64(requestPtr + 8, BigInt(process.argv[4]), true);
memory.setBigInt64(requestPtr + 16, BigInt(process.argv[5]), true);
memory.setUint32(requestPtr + 24, subject.length, true);
memory.setUint32(requestPtr + 28, evidence.length, true);
new Uint8Array(instance.exports.memory.buffer, requestPtr + 32, subject.length).set(subject);
new Uint8Array(instance.exports.memory.buffer, requestPtr + 32 + subject.length, evidence.length).set(evidence);
new Uint8Array(instance.exports.memory.buffer, namePtr, 14).set(
  new TextEncoder().encode("quota.admit-v1"),
);
const status = instance.exports.pulp_on_call(namePtr, 14, requestPtr, requestLength, outPtr, outPtr + 4);
if (status !== 0) throw new Error(`pulp_on_call returned ${status}`);
const responsePtr = memory.getUint32(outPtr, true);
const responseLen = memory.getUint32(outPtr + 4, true);
if (memory.getUint8(responsePtr) !== 0) throw new Error("unexpected ResultError");
const result = memory.getUint8(responsePtr + 1) !== 0;
const subjectLength = memory.getUint32(responsePtr + 2, true);
const evidenceLength = memory.getUint32(responsePtr + 6, true);
if (responseLen !== 10 + subjectLength + evidenceLength) throw new Error(`invalid response length ${responseLen}`);
const decodedSubject = new TextDecoder().decode(new Uint8Array(instance.exports.memory.buffer, responsePtr + 10, subjectLength));
const decodedEvidence = [...new Uint8Array(instance.exports.memory.buffer, responsePtr + 10 + subjectLength, evidenceLength)];
process.stdout.write(`${JSON.stringify({ result, events, subject: decodedSubject, evidence: decodedEvidence })}\n`);
