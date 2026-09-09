import fs from "node:fs";

const [wasmPath, vectorsPath, mode] = process.argv.slice(2);
if (!wasmPath || !vectorsPath) throw new Error("usage: aggregate-function-runner WASM VECTORS.json [--requests]");
const wasm = fs.readFileSync(wasmPath);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));

function encode(item) {
  const slice = item.slice.map(BigInt);
  const entries = item.map.map(([key, value]) => [BigInt(key), BigInt(value)]).sort((a, b) => a[0] < b[0] ? -1 : a[0] > b[0] ? 1 : 0);
  const result = Buffer.alloc(48 + slice.length * 8 + entries.length * 16);
  result.writeBigInt64LE(BigInt(item.record), 0);
  result.writeBigInt64LE(BigInt(item.array[0]), 8);
  result.writeBigInt64LE(BigInt(item.array[1]), 16);
  result.writeUInt32LE(48, 24); result.writeUInt32LE(slice.length, 28);
  const mapOffset = 48 + slice.length * 8;
  result.writeUInt32LE(mapOffset, 32); result.writeUInt32LE(entries.length, 36);
  result.writeBigInt64LE(BigInt(item.key), 40);
  slice.forEach((value, index) => result.writeBigInt64LE(value, 48 + index * 8));
  entries.forEach(([key, value], index) => { result.writeBigInt64LE(key, mapOffset + index * 16); result.writeBigInt64LE(value, mapOffset + index * 16 + 8); });
  return result;
}

async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(wasm, {}), api = instance.exports;
  const input = api.pulp_alloc(request.length), output = api.pulp_alloc(4), outputLength = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(request, input);
  const status = api.pulp_on_call(0, 0, input, request.length, output, outputLength);
  if (status !== 0) return { status };
  const view = new DataView(api.memory.buffer), pointer = view.getUint32(output, true);
  if (view.getUint32(outputLength, true) !== 8) throw new Error("aggregate.noncanonical_response");
  return { status, value: view.getBigInt64(pointer, true) };
}

const requests = Object.fromEntries(vectors.valid.map((item) => [item.name, encode(item).toString("hex")]));
const valid = {};
for (const item of vectors.valid) {
  const observed = await invoke(encode(item));
  if (observed.status !== 0 || observed.value !== BigInt(item.result)) throw new Error(`aggregate.mismatch:${item.name}`);
  valid[item.name] = observed.value.toString();
}
const base = encode(vectors.valid[0]);
const malformed = [Buffer.alloc(0), base.subarray(0, 16), Buffer.from(base)];
malformed[2].writeUInt32LE(47, 32); // map payload overlaps the fixed record/array/slice/key area
if (mode === "--requests") { console.log(JSON.stringify({ valid: requests, malformed: malformed.map((item) => item.toString("hex")) })); process.exit(0); }
if (mode === "--tsv") {
  for (const item of vectors.valid) { const response = Buffer.alloc(8); response.writeBigInt64LE(BigInt(item.result)); console.log(`valid\t${item.name}\t${requests[item.name]}\t${response.toString("hex")}`); }
  malformed.forEach((item, index) => console.log(`malformed\t${index}\t${item.toString("hex")}\t-`));
  process.exit(0);
}
for (const request of malformed) if ((await invoke(request)).status === 0) throw new Error("aggregate.malformed_accepted");
console.log(JSON.stringify({ valid, malformed: malformed.length }));
