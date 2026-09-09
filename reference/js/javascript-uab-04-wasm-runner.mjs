import fs from "node:fs";

const [wasmPath, vectorsPath, mode] = process.argv.slice(2);
if (!wasmPath || !vectorsPath) throw new Error("usage: javascript-uab-04-wasm-runner WASM VECTORS [--tsv]");
const wasm = fs.readFileSync(wasmPath), vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const encode = item => {
  const values = item.values.map(BigInt), bytes = Buffer.alloc(56 + values.length * 8);
  bytes.writeUInt32LE(56, 0); bytes.writeUInt32LE(values.length, 4);
  [item.index, item.replacement, item.appended, item.removeIndex, item.keepKey, item.removeKey].forEach((value, index) => bytes.writeBigInt64LE(BigInt(value), 8 + index * 8));
  values.forEach((value, index) => bytes.writeBigInt64LE(value, 56 + index * 8));
  return bytes;
};
async function invoke(request) {
  const {instance} = await WebAssembly.instantiate(wasm, {}), api = instance.exports;
  const input = api.pulp_alloc(request.length), output = api.pulp_alloc(4), length = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(request, input);
  try {
    const status = api.pulp_on_call(0, 0, input, request.length, output, length);
    if (status) return {status};
    const view = new DataView(api.memory.buffer), pointer = view.getUint32(output, true);
    return {status: 0, value: view.getBigInt64(pointer, true).toString()};
  } catch { return {status: -1}; }
}
const rows = [], valid = {}, malformed = {};
for (const item of vectors.valid) {
  const request = encode(item), result = await invoke(request);
  if (result.status || result.value !== item.result) throw new Error(`javascript.uab04.wasm:${item.name}`);
  valid[item.name] = result.value;
  const response = Buffer.alloc(8); response.writeBigInt64LE(BigInt(result.value)); rows.push(["valid", item.name, request.toString("hex"), response.toString("hex")]);
}
for (const item of vectors.malformed) {
  const request = encode(item);
  if (item.category === "collection-kind") request.writeUInt32LE(55, 0);
  const result = await invoke(request);
  if (result.status === 0) throw new Error(`javascript.uab04.malformed:${item.name}`);
  malformed[item.name] = true; rows.push(["malformed", item.name, request.toString("hex"), "-"]);
}
if (mode === "--tsv") rows.forEach(row => console.log(row.join("\t")));
else console.log(JSON.stringify({valid, malformed:Object.keys(malformed).length}));
