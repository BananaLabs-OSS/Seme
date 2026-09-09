import assert from "node:assert/strict";
import fs from "node:fs";
const [vectorsPath, auditPath] = process.argv.slice(2);
const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
const rows = fs.readFileSync(auditPath, "utf8").trim().split("\n").filter(Boolean).map(line => line.split("\t"));
assert.deepEqual(rows, [
  ...vectors.valid.map(item => ["valid", item.name, item.result]),
  ...vectors.malformed.map(item => ["malformed", item.name, "rejected"]),
]);
console.log("JavaScript UAB-04 pinned Pulp executed every named valid and malformed vector");
