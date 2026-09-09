import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript } from "./javascript-provider.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");

function lift(source, name) {
  return liftJavaScript({
    source,
    moduleG1,
    packagePath: `example.test/uab-javascript-fidelity/${name}`,
    revision: 1,
  });
}

const cases = [
  {
    name: "raw fixed-array indexing is not silently assigned checked Seme bounds",
    source: "/** @param {bigint[3]} values @param {bigint} index @returns {bigint} */ export function Pick(values, index) { return values[index]; }",
    diagnostic: /javascript\.raw_index_requires_adapter:1:\d+/,
  },
  {
    name: "raw slice indexing is not silently assigned checked Seme bounds",
    source: "/** @param {bigint[]} values @param {bigint} index @returns {bigint} */ export function Pick(values, index) { return values[index]; }",
    diagnostic: /javascript\.raw_index_requires_adapter:1:\d+/,
  },
  {
    name: "Number is not silently treated as canonical i64",
    source: "/** @returns {bigint} */ export function Value() { return 1; }",
    diagnostic: /javascript\.unsupported_expression:1:59/,
  },
  {
    name: "out-of-range BigInt is not silently wrapped",
    source: "/** @returns {bigint} */ export function Value() { return 9223372036854775808n; }",
    diagnostic: /javascript\.i64_literal_range:1:59/,
  },
  {
    name: "coercive equality is not canonical strict equality",
    source: "/** @param {string} value @returns {boolean} */ export function Empty(value) { return value == \"\"; }",
    diagnostic: /javascript\.unsupported_expression:1:\d+/,
  },
  {
    name: "truthiness is not silently treated as canonical Boolean",
    source: "/** @param {string} value @returns {boolean} */ export function Present(value) { if (value) return true; return false; }",
    diagnostic: /javascript\.unresolved_or_mistyped_identifier:1:\d+/,
  },
  {
    name: "null has no guessed Core meaning",
    source: "/** @returns {string} */ export function Value() { return null; }",
    diagnostic: /javascript\.unsupported_expression:1:59/,
  },
  {
    name: "undefined has no guessed Core meaning",
    source: "/** @returns {string} */ export function Value() { return undefined; }",
    diagnostic: /javascript\.unresolved_or_mistyped_identifier:1:59/,
  },
  {
    name: "exceptions are not guessed as canonical results",
    source: "/** @returns {string} */ export function Value() { throw new Error(\"no\"); }",
    diagnostic: /javascript\.unsupported_statement:1:\d+/,
  },
  {
    name: "async functions require an explicit future bridge profile",
    source: "/** @returns {string} */ export async function Value() { return \"ok\"; }",
    diagnostic: /javascript\.unsupported_function:1:\d+/,
  },
  {
    name: "generators require an explicit iterator bridge profile",
    source: "/** @returns {bigint} */ export function* Value() { yield 1n; }",
    diagnostic: /javascript\.unsupported_function:1:\d+/,
  },
  {
    name: "JavaScript lone surrogates are not canonical Unicode scalar text",
    source: "/** @returns {string} */ export function Value() { return \"\\ud800\"; }",
    diagnostic: /javascript\.non_scalar_string:1:59/,
  },
];

for (const item of cases) {
  test(item.name, () => {
    assert.throws(() => lift(item.source, item.name), item.diagnostic);
  });
}

test("BigInt-backed positive i64 boundary remains accepted as an explicit adaptation", () => {
  const maximum = lift("/** @returns {bigint} */ export function Value() { return 9223372036854775807n; }", "maximum-i64");
  assert.match(maximum, /uu 9223372036854775807/);
});
