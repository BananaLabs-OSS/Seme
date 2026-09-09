import assert from "node:assert/strict";
import { Seme } from "./seme-values.mjs";
import { canonicalValue, generatedSequences } from "./javascript-uab-11-vectors.mjs";

globalThis.Seme = Seme;
const source = process.argv[2] ?? new URL("../../fixtures/javascript-uab-11/application.js", import.meta.url).pathname;
const nativeModule = await import(new URL(`file://${source}`).href);
const Apply = nativeModule.Apply ?? nativeModule.Execute;
if (typeof Apply !== "function") throw new Error("javascript_uab11.entry_missing");
const ledger = [];
for (const sequence of generatedSequences()) {
  let state = sequence.initial;
  const steps = [];
  for (const command of sequence.commands) {
    const before = canonicalValue(state);
    const trace = [];
    const originalLog = console.log;
    console.log = (value) => trace.push(value);
    let outcome;
    try { outcome = Apply(state, command); } finally { console.log = originalLog; }
    assert.deepEqual(canonicalValue(state), before, "input state mutated");
    if (outcome.tag === "ok") {
      assert.deepEqual(trace, [true]);
      state = outcome.value.state;
      assert.equal(outcome.value.result, state.Counters.get(command.Key));
    } else {
      assert.deepEqual(trace, []);
      assert.ok([1n, 2n, 3n].includes(outcome.value));
    }
    steps.push({ command: canonicalValue(command), outcome: canonicalValue(outcome), trace });
  }
  ledger.push({ name: sequence.name, finalState: canonicalValue(state), steps });
}
process.stdout.write(`${JSON.stringify(ledger)}\n`);
