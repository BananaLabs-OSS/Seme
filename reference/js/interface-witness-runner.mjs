import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node interface-witness-runner.mjs <module.wasm>");
const { instance } = await WebAssembly.instantiate(await readFile(wasmPath), {});
const view = new DataView(instance.exports.memory.buffer);

function call(tag, state, value) {
  const request = instance.exports.pulp_alloc(24);
  const responsePointer = instance.exports.pulp_alloc(4);
  const responseLength = instance.exports.pulp_alloc(4);
  view.setBigUint64(request, tag, true);
  view.setBigInt64(request + 8, state, true);
  view.setBigInt64(request + 16, value, true);
  const status = instance.exports.pulp_on_call(0, 0, request, 24, responsePointer, responseLength);
  if (status !== 0) throw new Error(`provider status ${status}`);
  return view.getBigInt64(view.getUint32(responsePointer, true), true);
}

if (call(1n, 3n, 4n) !== 12n || call(2n, 3n, 4n) !== 7n) throw new Error("deterministic witness tags changed");
let rejected = false;
try { call(3n, 3n, 4n); } catch (error) { rejected = error instanceof WebAssembly.RuntimeError; }
if (!rejected) throw new Error("unknown witness tag accepted");
console.log(JSON.stringify({ witnesses: "deterministic", unknownTag: "trapped" }));
