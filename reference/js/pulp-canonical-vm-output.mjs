import fs from "node:fs";
import readline from "node:readline";

if (process.argv.length !== 3) throw new Error("usage: pulp-canonical-vm-output INPUT.jsonl");
const input = fs.createReadStream(process.argv[2]);
for await (const line of readline.createInterface({ input, crlfDelay: Infinity })) {
  if (!line) continue;
  const value = JSON.parse(line);
  if (typeof value.response !== "string" || !Array.isArray(value.effects) || value.effects.some((x) => typeof x !== "boolean")) {
    throw new Error("pulp_canonical_vm.output");
  }
  process.stdout.write(`${value.response}\t${JSON.stringify(value.effects)}\n`);
}
