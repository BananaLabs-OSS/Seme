import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript } from "./javascript-provider.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v14/module.g1", import.meta.url), "utf8");
const source = `/**
 * @param {string} left
 * @param {string} right
 * @returns {string}
 */
export function Join(left, right) {
  if (left === "") return right;
  return left + "λ" + right;
}`;

test("lifts typed native JavaScript text and total return control", () => {
  const first = liftJavaScript({ source, moduleG1, packagePath: "example.test/cross-text", revision: 1 });
  const second = liftJavaScript({ source, moduleG1, packagePath: "example.test/cross-text", revision: 1 });
  assert.equal(first, second);
  assert.match(first, /000000000000000000000000000090c0/);
  assert.match(first, /000000000000000000000000000090c3/);
});

test("rejects ambiguous JavaScript without boundary types", () => {
  assert.throws(
    () => liftJavaScript({ source: "export function Add(a, b) { return a + b; }", moduleG1, packagePath: "example.test/ambiguous", revision: 1 }),
    /javascript\.missing_jsdoc:1:8/,
  );
});

test("rejects unsupported coercive equality", () => {
  const coercive = source.replace('left === ""', 'left == ""');
  assert.throws(
    () => liftJavaScript({ source: coercive, moduleG1, packagePath: "example.test/coercive", revision: 1 }),
    /javascript\.unsupported_expression/,
  );
});

test("rejects JavaScript-only lone surrogate text", () => {
  const invalid = source.replace('left + "λ" + right', 'left + "\\ud800" + right');
  assert.throws(
    () => liftJavaScript({ source: invalid, moduleG1, packagePath: "example.test/surrogate", revision: 1 }),
    /javascript\.non_scalar_string/,
  );
});
