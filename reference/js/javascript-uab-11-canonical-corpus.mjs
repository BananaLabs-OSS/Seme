import fs from "node:fs";
import { Seme } from "./seme-values.mjs";
import { generatedSequences } from "./javascript-uab-11-vectors.mjs";

const [source, requestsPath, expectedPath] = process.argv.slice(2);
if (!source || !requestsPath || !expectedPath) throw new Error("usage: canonical-corpus SOURCE REQUESTS EXPECTED");
globalThis.Seme = Seme;
const { Apply } = await import(new URL(`file://${source}`).href);
const i64 = (value) => ({ kind: "i64", i64: value.toString() });
const stateValue = (state) => ({ kind: "record", fields: {
  Name: { kind: "text", text: state.Name },
  Values: { kind: "slice", items: state.Values.map(i64) },
  Counters: { kind: "map", value_type: "i64", entries: [...state.Counters].sort(([a], [b]) => a < b ? -1 : 1).map(([key, value]) => ({ key: i64(key), value: i64(value) })) },
} });
const commandValue = (command) => ({ kind: "record", fields: {
  Key: i64(command.Key), Index: i64(command.Index), Delta: i64(command.Delta), Amount: i64(command.Amount), Scale: { kind: "bool", bool: command.Scale },
} });
const outcomeValue = (outcome) => outcome.tag === "error"
  ? { kind: "result", variant: "error", payload: i64(outcome.value) }
  : { kind: "result", variant: "ok", payload: { kind: "transition", state: stateValue(outcome.value.state), result: i64(outcome.value.result) } };
const requests = [], expected = [];
for (const sequence of generatedSequences()) {
  let state = sequence.initial;
  for (const command of sequence.commands) {
    requests.push({ arguments: [stateValue(state), commandValue(command)], capabilities: ["observability.log"] });
    const effects = [];
    const log = console.log; console.log = (value) => effects.push({ capability: "observability.log", value });
    let outcome;
    try { outcome = Apply(state, command); } finally { console.log = log; }
    expected.push({ value: outcomeValue(outcome), effects });
    if (outcome.tag === "ok") state = outcome.value.state;
  }
}
fs.writeFileSync(requestsPath, `${requests.map(JSON.stringify).join("\n")}\n`);
fs.writeFileSync(expectedPath, `${expected.map(JSON.stringify).join("\n")}\n`);
