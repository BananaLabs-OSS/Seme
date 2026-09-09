import assert from "node:assert/strict";
import fs from "node:fs";

const [vectorsPath, ...paths] = process.argv.slice(2);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const expectedValid = Object.fromEntries(vectors.valid.map(item => [item.name, String(item.result)]));
const expectedRejected = Object.fromEntries(vectors.malformed.map(item => [item.name, true]));
for (const path of paths) {
  const observed = JSON.parse(fs.readFileSync(path, "utf8"));
  const valid = Object.fromEntries(Object.entries(observed.valid).map(([name, value]) => [name, value?.kind === "i64" ? value.i64 : String(value)]));
  assert.deepEqual(valid, expectedValid, `${path}: valid observations`);
  assert.equal(observed.malformed, vectors.malformed.length, `${path}: malformed count`);
  assert.deepEqual(observed.rejected, expectedRejected, `${path}: named malformed observations`);
}
console.log("Go UAB-04 observations: every valid result and named malformed rejection agrees");
