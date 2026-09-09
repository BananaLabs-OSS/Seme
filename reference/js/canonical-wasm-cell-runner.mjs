import fs from "node:fs";
import { WASI } from "node:wasi";

if (process.argv.length < 5 || process.argv.length > 6) {
  throw new Error("usage: canonical-wasm-cell-runner CELL.wasm PROGRAM.seme REQUESTS.hex [--deny]");
}

const [wasmPath, graphPath, requestsPath] = process.argv.slice(2);
const deny = process.argv[5] === "--deny";
const observations = [];
const wasi = new WASI({ version: "preview1", args: [], env: {} });
const imports = {
  ...wasi.getImportObject(),
  pulp: {
    log_bool(value) {
	  if (deny) return 99;
      observations.push(value !== 0);
      return 0;
    },
  },
};
const module = await WebAssembly.compile(fs.readFileSync(wasmPath));
const instance = await WebAssembly.instantiate(module, imports);
// The reusable cell is normally a reactor (`_initialize`), as required by
// Pulp. Retaining command-module support makes this diagnostic runner useful
// when inspecting an ordinary Go wasip1 build as well.
if (typeof instance.exports._start === "function") wasi.start(instance);
else wasi.initialize(instance);
const e = instance.exports;

function allocate(bytes) {
  if (bytes.length === 0) return 0;
  const ptr = e.pulp_alloc(bytes.length);
  if (!ptr) throw new Error("canonical_vm.alloc");
  new Uint8Array(e.memory.buffer, ptr, bytes.length).set(bytes);
  return ptr;
}
function diagnostic(code) {
  const ptr = e.pulp_on_call_error_ptr();
  const len = e.pulp_on_call_error_len();
  const message = len ? Buffer.from(e.memory.buffer, ptr, len).toString("utf8") : "";
  throw new Error(`canonical_vm.code_${code}:${message}`);
}

const graph = fs.readFileSync(graphPath);
const graphPtr = allocate(graph);
const initCode = e.pulp_init(graphPtr, graph.length);
e.pulp_free(graphPtr, graph.length);
if (initCode) diagnostic(initCode);

const name = Buffer.from("seme.evaluate.v1");
for (const line of fs.readFileSync(requestsPath, "utf8").trim().split("\n")) {
  if (!line) continue;
  const request = Buffer.from(line.trim(), "hex");
  const namePtr = allocate(name);
  const requestPtr = allocate(request);
  const outPtr = e.pulp_alloc(8);
  const before = observations.length;
  const code = e.pulp_on_call(namePtr, name.length, requestPtr, request.length, outPtr, outPtr + 4);
  e.pulp_free(namePtr, name.length);
  e.pulp_free(requestPtr, request.length);
  if (code) diagnostic(code);
  const view = new DataView(e.memory.buffer);
  const responsePtr = view.getUint32(outPtr, true);
  const responseLen = view.getUint32(outPtr + 4, true);
  const response = Buffer.from(new Uint8Array(e.memory.buffer, responsePtr, responseLen));
  e.pulp_free(responsePtr, responseLen);
  e.pulp_free(outPtr, 8);
  process.stdout.write(`${response.toString("hex")}\t${JSON.stringify(observations.slice(before))}\n`);
}
if (e.pulp_shutdown() !== 0) throw new Error("canonical_vm.shutdown");
