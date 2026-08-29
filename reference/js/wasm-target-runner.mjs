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
    },
  },
});
const result = instance.exports.admit(
  BigInt(process.argv[3]),
  BigInt(process.argv[4]),
  BigInt(process.argv[5]),
) !== 0;
process.stdout.write(`${JSON.stringify({ result, events })}\n`);
