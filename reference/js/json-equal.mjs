import assert from "node:assert/strict";
import fs from "node:fs";
const [left, right] = process.argv.slice(2);
if (!left || !right) throw new Error("usage: json-equal LEFT.json RIGHT.json");
assert.deepEqual(JSON.parse(fs.readFileSync(left,"utf8")), JSON.parse(fs.readFileSync(right,"utf8")));
