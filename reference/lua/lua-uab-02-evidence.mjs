import assert from "node:assert/strict";
import fs from "node:fs";

const report = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
assert.equal(report.profile, "seme.uab/v1");
assert.equal(report.cell, "UAB-02");
assert.equal(report.status, "incomplete");
const required = ["i64", "boolean", "text", "bytes", "records", "result", "option", "fixed_arrays", "slices", "runtime_maps"];
assert.deepEqual(Object.keys(report.families), required);
for (const family of required) {
  const evidence = report.families[family];
  assert.deepEqual(Object.keys(evidence), ["lift", "native_adapter", "canonical_validation", "projection_round_trip", "rejection", "target_parity"]);
  assert.equal(evidence.lift, true);
  assert.equal(evidence.native_adapter, true);
  assert.equal(evidence.canonical_validation, true);
  assert.equal(evidence.projection_round_trip, true);
  assert.equal(evidence.rejection, true);
  assert.equal(evidence.target_parity, ["bytes", "result", "option"].includes(family));
}
console.log(`Lua UAB-02 evidence: 10/10 families structurally bridged; 3/10 target-parity families certified; cell incomplete`);
