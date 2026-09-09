import fs from "node:fs";

const [input, mode, output] = process.argv.slice(2);
if (!input || !mode || !output) throw new Error("usage: forge INPUT MODE OUTPUT");
let graph = fs.readFileSync(input, "utf8");
const i64 = graph.match(/^en ([0-9a-f]{32}) 00000000000000000000000000009010 /m)?.[1];
if (!i64) throw new Error("javascript_uab11.i64_missing");
const mutate = (schema, field) => {
  const expression = new RegExp(`(en [0-9a-f]{32} ${schema} [\\s\\S]*?fi ${field} rf )[0-9a-f]{32}(?=\\n(?:en |$))`);
  // Field lines may be followed by more fields, so bound the entity first.
  const start = graph.search(new RegExp(`^en [0-9a-f]{32} ${schema} `, "m"));
  if (start < 0) throw new Error(`javascript_uab11.schema_missing:${schema}`);
  const endOffset = graph.slice(start + 1).search(/^en /m);
  const end = endOffset < 0 ? graph.length : start + 1 + endOffset;
  const block = graph.slice(start, end);
  const changed = block.replace(new RegExp(`(fi ${field} rf )[0-9a-f]{32}`), `$1${i64}`);
  if (changed === block) throw new Error(`javascript_uab11.field_missing:${field}`);
  graph = graph.slice(0, start) + changed + graph.slice(end);
};
if (mode === "map-option-type") mutate("0000000000000000000000000000a044", "000000000000000000000000000a0442");
else if (mode === "transition-type") mutate("0000000000000000000000000000a005", "000000000000000000000000000a0050");
else if (mode === "effect-authority") mutate("00000000000000000000000000000015", "00000000000000000000000000000151");
else if (mode === "erase-stateful-call") {
  const pattern = /(en [0-9a-f]{32} )0000000000000000000000000000a035( 1 2)/;
  const changed = graph.replace(pattern, "$1" + "0000000000000000000000000000a024" + "$2");
  if (changed === graph) throw new Error("javascript_uab11.stateful_call_missing");
  graph = changed;
}
else throw new Error(`javascript_uab11.unknown_forgery:${mode}`);
fs.writeFileSync(output, graph);
