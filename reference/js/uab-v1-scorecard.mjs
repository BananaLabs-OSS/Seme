import { readFile } from "node:fs/promises";

const path = process.argv[2];
if (!path) throw new Error("usage: node uab-v1-scorecard.mjs SCORECARD.json");
const scorecard = JSON.parse(await readFile(path, "utf8"));
const languages = ["go", "javascript", "lua"];
const evidence = ["lift", "native_parity", "target_parity", "projection_round_trip", "rejection"];
const ids = Array.from({ length: 12 }, (_, index) => `UAB-${String(index + 1).padStart(2, "0")}`);

if (scorecard.profile !== "seme.uab/v1") throw new Error("wrong profile identity");
if (JSON.stringify(scorecard.evidence) !== JSON.stringify(evidence)) throw new Error("evidence denominator changed");
if (JSON.stringify(Object.keys(scorecard.languages)) !== JSON.stringify(languages)) throw new Error("language denominator changed");

const result = { profile: scorecard.profile, passed: 0, total: languages.length * ids.length, languages: {} };
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
