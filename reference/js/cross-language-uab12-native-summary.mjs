import fs from "node:fs";
import { generatedSequences } from "./javascript-uab-11-vectors.mjs";

if (process.argv.length !== 3) throw new Error("usage: cross-language-uab12-native-summary EXPECTED.jsonl");
const lines = fs.readFileSync(process.argv[2], "utf8").trim().split("\n").filter(Boolean).map(JSON.parse);
const sequences = generatedSequences();
if (lines.length !== sequences.length * 16) throw new Error("cross_language_summary.cardinality");
const summary = { success: 0, missing: 0, index: 0, negative: 0, traces: 0, observations: [] };
let cursor = 0;
for (const sequence of sequences) {
  for (let step = 1; step <= sequence.commands.length; step++) {
    const observation = lines[cursor++];
    const outcome = observation.value;
    if (outcome?.kind !== "result" || !["ok", "error"].includes(outcome.variant)) throw new Error("cross_language_summary.result");
    const trace = observation.effects.map((effect) => {
      if (effect.capability !== "observability.log" || typeof effect.value !== "boolean") throw new Error("cross_language_summary.effect");
      return effect.value;
    });
    let result;
    if (outcome.variant === "ok") {
      if (outcome.payload?.kind !== "transition" || outcome.payload.result?.kind !== "i64" || trace.length !== 1 || trace[0] !== true) throw new Error("cross_language_summary.success");
      result = outcome.payload.result.i64;
      summary.success++; summary.traces++;
    } else {
      if (outcome.payload?.kind !== "i64" || trace.length !== 0) throw new Error("cross_language_summary.error");
      result = outcome.payload.i64;
      if (result === "1") summary.missing++;
      else if (result === "2") summary.index++;
      else if (result === "3") summary.negative++;
      else throw new Error("cross_language_summary.error_code");
    }
    summary.observations.push({ name: `${sequence.name}-${step}`, outcome: outcome.variant, result, trace });
  }
}
process.stdout.write(`${JSON.stringify(summary)}\n`);
