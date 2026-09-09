import fs from "node:fs";

const [wasmPath, vectorsPath, mode] = process.argv.slice(2);
if (!wasmPath || !vectorsPath) throw new Error("usage: javascript-uab-03-wasm-runner WASM VECTORS [--tsv]");
const wasm = fs.readFileSync(wasmPath), vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const encode = (args) => {
  const request = Buffer.alloc(33);
  request.writeBigInt64LE(BigInt(args[0]), 0); request.writeBigInt64LE(BigInt(args[1]), 8);
  if (typeof args[2] !== "boolean") throw new TypeError("javascript.boolean_boundary");
  request[16] = args[2] ? 1 : 0;
  request.writeBigInt64LE(BigInt(args[3]), 17); request.writeBigInt64LE(BigInt(args[4]), 25);
  return request;
};
async function invoke(request) {
  const { instance } = await WebAssembly.instantiate(wasm, {}), api = instance.exports;
  const input = api.pulp_alloc(request.length), output = api.pulp_alloc(4), length = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(request, input);
  let status;
  try { status = api.pulp_on_call(0, 0, input, request.length, output, length); }
  catch { return { status: -1 }; }
  if (status !== 0) return { status };
  const view = new DataView(api.memory.buffer), pointer = view.getUint32(output, true);
  if (view.getUint32(length, true) !== 8) throw new Error("javascript.uab03.response_size");
  return { status, value: view.getBigInt64(pointer, true) };
}
const valid = {};
for (const item of vectors.valid) {
  const request = encode(item.arguments), observed = await invoke(request);
  if (observed.status !== 0 || observed.value.toString() !== item.result) throw new Error(`javascript.uab03.wasm_mismatch:${item.name}`);
  valid[item.name] = observed.value.toString();
  if (mode === "--tsv") { const response=Buffer.alloc(8); response.writeBigInt64LE(BigInt(item.result)); console.log(`valid\t${item.name}\t${request.toString("hex")}\t${response.toString("hex")}`); }
}
const andHazard=["9","4",true,"1","0"], orHazard=["9","4",false,"0","1"];
const malformed = [Buffer.alloc(0), Buffer.alloc(32), Buffer.concat([encode(vectors.valid[0].arguments), Buffer.from([0])]), encode(vectors.valid[0].arguments), encode(andHazard), encode(orHazard)];
malformed[3][16] = 2;
for (const [index, request] of malformed.entries()) {
  if ((await invoke(request)).status === 0) throw new Error(`javascript.uab03.malformed_accepted:${index}`);
  if (mode === "--tsv") console.log(`malformed\t${index}\t${request.toString("hex")}\t-`);
}
if (mode !== "--tsv") console.log(JSON.stringify({ valid, malformed: malformed.length }));
