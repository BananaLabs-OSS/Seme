import { readFile } from "node:fs/promises";

const path = process.argv[2];
if (!path) throw new Error("usage: check-upb-v1-scorecard.mjs SCORECARD.json");
const scorecard = JSON.parse(await readFile(path, "utf8"));
const languages = ["go", "javascript", "lua"];
const evidence = ["native_project", "project_lift", "canonical_parity", "resolution_fidelity", "target_parity", "project_round_trip", "atomic_rejection"];
const ids = Array.from({ length: 12 }, (_, index) => `UPB-${String(index + 1).padStart(2, "0")}`);

if (scorecard.profile !== "seme.upb/v1") throw new Error("wrong profile identity");
if (JSON.stringify(scorecard.evidence) !== JSON.stringify(evidence)) throw new Error("evidence denominator changed");
if (JSON.stringify(Object.keys(scorecard.languages)) !== JSON.stringify(languages)) throw new Error("language denominator changed");

const result = { profile: scorecard.profile, passed: 0, total: languages.length * ids.length, evidence_per_cell: evidence.length, languages: {} };
for (const language of languages) {
  const cells = scorecard.languages[language];
  if (JSON.stringify(Object.keys(cells)) !== JSON.stringify(ids)) throw new Error(`${language} capability denominator changed`);
  let passed = 0;
  for (const id of ids) {
    const claims = cells[id];
    if (!Array.isArray(claims) || new Set(claims).size !== claims.length || claims.some((claim) => !evidence.includes(claim))) {
      throw new Error(`${language} ${id} has invalid evidence`);
    }
    if (evidence.every((claim) => claims.includes(claim))) passed += 1;
  }
  result.languages[language] = { passed, total: ids.length };
  result.passed += passed;
}
console.log(JSON.stringify(result));

