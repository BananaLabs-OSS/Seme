import assert from "node:assert/strict";
import fs from "node:fs";
import { pathToFileURL } from "node:url";

const sourcePath = process.argv[2];
const vectorsPath = process.argv[3];
if (!sourcePath && !vectorsPath) {
  const { Run } = await import("../../fixtures/javascript-uab-01/application.js");
  assert.equal(Run(0n, 0n), 0n);
  assert.equal(Run(7n, 5n), 12n);
  assert.equal(Run(-7n, 5n), -2n);
} else {
  if (!sourcePath || !vectorsPath) throw new Error("usage: javascript-package-native-runner SOURCE VECTORS");
  const module = await import(`${pathToFileURL(sourcePath)}?uab01=${Date.now()}`);
  if (typeof module.Run !== "function") throw new Error("javascript.uab01.entry_missing");
  const vectors = JSON.parse(fs.readFileSync(vectorsPath, "utf8"));
  const valid = {};
  for (const item of vectors.valid) {
    const args = item.arguments.map((value) => BigInt(value.i64));
    const observed = module.Run(...args);
    assert.equal(typeof observed, "bigint");
    assert.equal(observed.toString(), item.result.i64, item.name);
    valid[item.name] = observed.toString();
  }
  process.stdout.write(`${JSON.stringify({ valid })}\n`);
}
