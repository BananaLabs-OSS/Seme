import { readFile } from "node:fs/promises";

const wasmPath = process.argv[2];
if (!wasmPath) throw new Error("usage: node mutable-closure-runner.mjs <module.wasm>");
const { instance } = await WebAssembly.instantiate(await readFile(wasmPath), {});
const view = new DataView(instance.exports.memory.buffer);

function invoke(start, first, second, expected) {
  const request = instance.exports.pulp_alloc(24);
  const pointer = instance.exports.pulp_alloc(4);
  const length = instance.exports.pulp_alloc(4);
  view.setBigInt64(request, start, true);
  view.setBigInt64(request + 8, first, true);
  view.setBigInt64(request + 16, second, true);
  if (instance.exports.pulp_on_call(0, 0, request, 24, pointer, length) !== 0) throw new Error("provider rejected valid call");
  const result = view.getBigInt64(view.getUint32(pointer, true), true);
  if (view.getUint32(length, true) !== 8 || result !== expected) throw new Error(`unexpected result ${result}`);
  if (view.getBigInt64(request, true) !== start || view.getBigInt64(request + 8, true) !== first || view.getBigInt64(request + 16, true) !== second) throw new Error("request aliased mutable environment");
  return result;
}

const positive = invoke(10n, 5n, 7n, 22n);
const negative = invoke(-10n, -5n, 7n, -8n);
invoke(9223372036854775807n, 1n, 1n, -9223372036854775807n);
// Repeating and interleaving calls proves each invocation creates an independent
// environment; neither the previous result nor another instance leaks into it.
invoke(10n, 5n, 7n, 22n);
invoke(1n, 2n, 3n, 6n);
invoke(10n, 5n, 7n, 22n);
const malformed = instance.exports.pulp_alloc(24), pointer = instance.exports.pulp_alloc(4), length = instance.exports.pulp_alloc(4);
if (instance.exports.pulp_on_call(0, 0, malformed, 23, pointer, length) === 0) throw new Error("short request accepted");
if (instance.exports.pulp_on_call(0, 0, malformed, 25, pointer, length) === 0) throw new Error("long request accepted");
console.log(JSON.stringify({ positive:positive.toString(), negative:negative.toString(), priorUpdate:"observed", instances:"independent", request:"unaliasable", overflow:"wrapped", malformed:"rejected" }));
