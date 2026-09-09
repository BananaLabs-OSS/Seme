import fs from "node:fs";

const [wasmPath, vectorsPath, mode] = process.argv.slice(2);
if (!wasmPath || !vectorsPath) throw new Error("usage: scalar-function-runner WASM VECTORS.json [--tsv]");
const wasm = fs.readFileSync(wasmPath), vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const utf8 = new TextEncoder(), decode = new TextDecoder("utf-8", { fatal: true });
function encode(item) {
  const label = utf8.encode(item.label), request = Buffer.alloc(17 + label.length);
  request.writeBigInt64LE(BigInt(item.value), 0); request[8] = item.enabled ? 1 : 0;
  request.writeUInt32LE(17, 9); request.writeUInt32LE(label.length, 13); request.set(label, 17); return request;
}
async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(wasm, {}), api = instance.exports;
  const input = api.pulp_alloc(request.length), output = api.pulp_alloc(4), outputLength = api.pulp_alloc(4), memory = new Uint8Array(api.memory.buffer);
  memory.set(request, input); const status = api.pulp_on_call(0, 0, input, request.length, output, outputLength);
  if (status !== 0) return { status };
  const view = new DataView(api.memory.buffer), pointer = view.getUint32(output, true), length = view.getUint32(outputLength, true);
  const response = memory.slice(pointer, pointer + length);
  return { status, value: decode.decode(response), response: Buffer.from(response).toString("hex") };
}
const base = encode(vectors.valid[0]), malformed = [Buffer.from(base), Buffer.from(base), Buffer.concat([base, Buffer.from([0])])];
malformed[0][8] = 2;
malformed[1] = Buffer.alloc(19); malformed[1].writeBigInt64LE(-7n, 0); malformed[1][8] = 1; malformed[1].writeUInt32LE(17, 9); malformed[1].writeUInt32LE(2, 13); malformed[1].set([0xc3, 0x28], 17);
if (mode === "--tsv") {
  for (const item of vectors.valid) { const observed = await invoke(encode(item)); console.log(`valid\t${item.name}\t${encode(item).toString("hex")}\t${observed.response}`); }
  malformed.forEach((item, index) => console.log(`malformed\t${index}\t${item.toString("hex")}\t-`)); process.exit(0);
}
const valid = {};
for (const item of vectors.valid) { const observed = await invoke(encode(item)); if (observed.status || observed.value !== item.result) throw new Error(`scalar.mismatch:${item.name}`); valid[item.name] = observed.value; }
for (const request of malformed) if ((await invoke(request)).status === 0) throw new Error("scalar.malformed_accepted");
console.log(JSON.stringify({ valid, malformed: malformed.length }));
