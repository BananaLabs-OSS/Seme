import fs from "node:fs";

const [mode, path] = process.argv.slice(2);
if (!mode || !path) throw new Error("usage: javascript-uab-04-vectors MODE VECTORS.json");
const vectors = JSON.parse(fs.readFileSync(path, "utf8"));
const i64 = value => ({kind: "i64", i64: String(value)});
const slice = values => ({kind: "slice", items: values.map(i64)});
const args = (item, wrongIndex = false) => [
  slice(item.values), wrongIndex ? {kind: "text", text: item.index} : i64(item.index),
  i64(item.replacement), i64(item.appended), i64(item.removeIndex), i64(item.keepKey), i64(item.removeKey),
];

if (mode === "canonical") {
  console.log(JSON.stringify({
    valid: vectors.valid.map(item => ({name: item.name, arguments: args(item), result: i64(item.result)})),
    malformed: vectors.malformed.map(item => ({name: item.name, arguments: args(item, item.category === "index-kind")})),
  }));
} else if (mode === "expected") {
  console.log(JSON.stringify({valid: Object.fromEntries(vectors.valid.map(item => [item.name, item.result])), malformed: vectors.malformed.length}));
} else throw new Error("javascript.uab04.vector_mode");
