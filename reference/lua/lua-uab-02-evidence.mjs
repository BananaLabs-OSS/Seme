import assert from "node:assert/strict";
import fs from "node:fs";

const report = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
assert.equal(report.profile, "seme.uab/v1");
assert.equal(report.cell, "UAB-02");
assert.equal(report.status, "complete");
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
  assert.equal(evidence.target_parity, true);
}
assert.deepEqual(report.blockers, []);
console.log(`Lua UAB-02 evidence: all 10 families pass native, canonical, target, projection, and rejection evidence`);
