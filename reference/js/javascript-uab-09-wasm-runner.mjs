import fs from "node:fs";

const [wasmPath, vectorsPath, mode = "json"] = process.argv.slice(2);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const bytes = fs.readFileSync(wasmPath);

function hex(values) {
  return Buffer.from(values).toString("hex");
}

const valid = {};
const rows = [];
for (const vector of vectors.valid) {
  const trace = [];
  const { instance } = await WebAssembly.instantiate(bytes, {
    pulp: { log_bool(value) { trace.push(value !== 0); return 0; } },
  });
  const api = instance.exports;
  const provider = new TextEncoder().encode("seme.function-v1");
  const request = Uint8Array.from(vector.arguments, value => value ? 1 : 0);
  const pp = api.pulp_alloc(provider.length), rp = api.pulp_alloc(request.length), op = api.pulp_alloc(8);
  let memory = new Uint8Array(api.memory.buffer);
  memory.set(provider, pp); memory.set(request, rp);
  const status = api.pulp_on_call(pp, provider.length, rp, request.length, op, op + 4);
  if (status !== 0) throw Error(`status:${vector.name}:${status}`);
  const view = new DataView(api.memory.buffer), outp = view.getUint32(op, true), outn = view.getUint32(op + 4, true);
  memory = new Uint8Array(api.memory.buffer);
  const response = memory.slice(outp, outp + outn);
  const result = response.length === 1 && response[0] !== 0;
  if (result !== vector.result || JSON.stringify(trace) !== JSON.stringify(vector.trace)) throw Error(`mismatch:${vector.name}`);
  valid[vector.name] = { result, trace };
  rows.push([vector.name, hex(request), hex(response), vector.trace.join(",")].join("\t"));
}
process.stdout.write(mode === "tsv" ? `${rows.join("\n")}\n` : JSON.stringify({ valid }));
