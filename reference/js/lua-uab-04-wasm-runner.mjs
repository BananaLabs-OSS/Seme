import fs from "node:fs";

const [wasmPath, vectorsPath, mode] = process.argv.slice(2);
if (!wasmPath || !vectorsPath) throw new Error("usage: lua-uab-04-wasm-runner WASM VECTORS [--tsv]");
const wasm = fs.readFileSync(wasmPath);
const corpus = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const validVectors = corpus.valid.filter(item => item.entry === "Evaluate");
const malformedVectors = corpus.malformed.filter(item => item.entry === "Evaluate");

function encode(item) {
  if (item.arguments.length !== 9 || item.arguments.some(value => value.kind !== "i64")) throw new Error(`lua.uab04.abi:${item.name}`);
  const request = Buffer.alloc(72);
  item.arguments.forEach((value, index) => request.writeBigInt64LE(BigInt(value.i64), index * 8));
  return request;
}

async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(wasm, {});
  const api = instance.exports;
  const input = api.pulp_alloc(request.length), output = api.pulp_alloc(4), length = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(request, input);
  try {
    const status = api.pulp_on_call(0, 0, input, request.length, output, length);
    if (status) return { status };
    const view = new DataView(api.memory.buffer), pointer = view.getUint32(output, true);
    return { status: 0, value: view.getBigInt64(pointer, true).toString() };
  } catch {
    return { status: -1 };
  }
}

const rows = [], valid = {}, malformed = {};
for (const item of validVectors) {
  const request = encode(item), result = await invoke(request);
  if (result.status || result.value !== item.result) throw new Error(`lua.uab04.wasm:${item.name}`);
  valid[item.name] = result.value;
  const response = Buffer.alloc(8); response.writeBigInt64LE(BigInt(result.value));
  rows.push(["valid", item.name, request.toString("hex"), response.toString("hex")]);
}
for (const item of malformedVectors) {
  const request = encode(item), result = await invoke(request);
  if (result.status === 0) throw new Error(`lua.uab04.malformed:${item.name}`);
  malformed[item.name] = true;
  rows.push(["malformed", item.name, request.toString("hex"), "-"]);
}
if (mode === "--tsv") rows.forEach(row => console.log(row.join("\t")));
else console.log(JSON.stringify({ valid, malformed }));
