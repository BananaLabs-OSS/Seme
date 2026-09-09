import assert from "node:assert/strict";
import fs from "node:fs";

const read = (path) => fs.readFileSync(path, "utf8").trim().split("\n").filter(Boolean).map(JSON.parse);
const [expected, actual] = process.argv.slice(2);
assert.deepStrictEqual(read(actual), read(expected));
