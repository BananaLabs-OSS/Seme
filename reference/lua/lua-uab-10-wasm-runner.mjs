import fs from "node:fs";

const wasm = fs.readFileSync(process.argv[2]);
const cases = [["false", Buffer.from([0]), 0], ["true", Buffer.from([1]), 1], ["missing", Buffer.alloc(0)], ["extra", Buffer.from([0, 1])], ["wrong-type", Buffer.from([2])]];
for (const [name, request, expected] of cases) {
  const { instance } = await WebAssembly.instantiate(wasm, {}), api = instance.exports;
  if (api.pulp_init(0, 0) !== 0) throw new Error("lua.uab10.init");
  const input = api.pulp_alloc(request.length), outPointer = api.pulp_alloc(4), outLength = api.pulp_alloc(4);
  new Uint8Array(api.memory.buffer).set(request, input);
  let status = 255;
  try { status = api.pulp_on_call(0, 0, input, request.length, outPointer, outLength); } catch {}
  if (expected === undefined) {
    if (status === 0) throw new Error(`lua.uab10.accepted:${name}`);
    console.log(JSON.stringify({ name, rejected: true }));
    continue;
  }
  if (status !== 0) throw new Error(`lua.uab10.rejected:${name}`);
  const view = new DataView(api.memory.buffer), pointer = view.getUint32(outPointer, true), length = view.getUint32(outLength, true);
  if (length !== 1 || new Uint8Array(api.memory.buffer)[pointer] !== expected) throw new Error(`lua.uab10.result:${name}`);
  console.log(JSON.stringify({ name, response: expected === 0 ? "00" : "01" }));
}
