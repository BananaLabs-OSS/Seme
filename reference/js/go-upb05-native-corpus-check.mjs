import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";

const [requestsPath, expectedPath] = process.argv.slice(2);
if (!requestsPath || !expectedPath) throw new Error("usage: go-upb05-native-corpus-check REQUESTS EXPECTED");

const lines = async (path) => (await readFile(path, "utf8")).trimEnd().split("\n").map((line) => JSON.parse(line));
const requests = await lines(requestsPath);
const expected = await lines(expectedPath);
assert.equal(requests.length, 2058);
assert.equal(expected.length, requests.length);

for (let index = 0; index < requests.length; index += 1) {
  const request = requests[index];
  assert.deepEqual(request.capabilities, ["observability.log"]);
  assert.equal(request.arguments.length, 3);
  const [configuration, state, command] = request.arguments;
  assert.equal(configuration.kind, "record");
  assert.deepEqual(Object.keys(configuration.fields).sort(), ["Limit", "NamePrefix", "UseDefaultLimit"]);
  assert.equal(configuration.fields.NamePrefix.kind, "text");
  assert.equal(configuration.fields.Limit.kind, "i64");
  assert.equal(configuration.fields.UseDefaultLimit.kind, "bool");
  assert.equal(state.kind, "record");
  assert.deepEqual(Object.keys(state.fields).sort(), ["Counters", "Name", "Values"]);
  assert.equal(command.kind, "record");
  assert.deepEqual(Object.keys(command.fields).sort(), ["Amount", "Delta", "Index", "Key", "Scale"]);
  assert.equal(expected[index].value.kind, "result");
  assert.ok(Array.isArray(expected[index].effects));
  for (const effect of expected[index].effects) assert.deepEqual(effect, { capability: "observability.log", value: true });
}

for (let index = 0; index < 2048; index += 1) {
  const fields = requests[index].arguments[0].fields;
  assert.equal(fields.NamePrefix.text, "sequence");
  assert.equal(fields.UseDefaultLimit.bool, true);
}

const boundary = requests.slice(2048).map((request) => request.arguments[0].fields);
assert.equal(boundary.length, 10);
assert.ok(boundary.some((fields) => fields.NamePrefix.text === "世界" && fields.UseDefaultLimit.bool));
assert.ok(boundary.some((fields) => fields.Limit.i64 === "1"));
assert.ok(boundary.some((fields) => fields.Limit.i64 === "4096"));
assert.ok(boundary.some((fields) => fields.NamePrefix.text === ""));
assert.ok(boundary.some((fields) => fields.Limit.i64 === "-1"));
assert.ok(boundary.some((fields) => fields.Limit.i64 === "4097"));

process.stdout.write(JSON.stringify({ profile: "go-upb05-native-v1", cases: requests.length, base_cases: 2048, configuration_cases: 10, arguments: 3 }) + "\n");
