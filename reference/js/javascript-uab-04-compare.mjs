import assert from "node:assert/strict";
import fs from "node:fs";

const [expectedPath, ...observedPaths] = process.argv.slice(2);
const expected = JSON.parse(fs.readFileSync(expectedPath, "utf8"));
for (const path of observedPaths) {
  const value = JSON.parse(fs.readFileSync(path, "utf8"));
  const normalized = {valid: Object.fromEntries(Object.entries(value.valid).map(([name, result]) => [name, result?.kind === "i64" ? result.i64 : String(result)])), malformed: value.malformed};
  assert.deepEqual(normalized, expected, path);
}
console.log("JavaScript UAB-04 exact named observations agree");
