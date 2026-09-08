import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript } from "./javascript-provider.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v14/module.g1", import.meta.url), "utf8");
const moduleV15G1 = fs.readFileSync(new URL("../../modules/execution/v15/module.g1", import.meta.url), "utf8");
const moduleV16G1 = fs.readFileSync(new URL("../../modules/execution/v16/module.g1", import.meta.url), "utf8");
const moduleV18G1 = fs.readFileSync(new URL("../../modules/execution/v18/module.g1", import.meta.url), "utf8");
const moduleV19G1 = fs.readFileSync(new URL("../../modules/execution/v19/module.g1", import.meta.url), "utf8");
const moduleV20G1 = fs.readFileSync(new URL("../../modules/execution/v20/module.g1", import.meta.url), "utf8");
const moduleV21G1 = fs.readFileSync(new URL("../../modules/execution/v21/module.g1", import.meta.url), "utf8");
const moduleV22G1 = fs.readFileSync(new URL("../../modules/execution/v22/module.g1", import.meta.url), "utf8");
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

test("repeated reads retain one semantic identity", () => {
  const repeated = source.replace('left + "λ" + right', 'left + "λ" + left');
  const canonical = liftJavaScript({ source: repeated, moduleG1, packagePath: "example.test/repeated", revision: 1 });
  const ids = [...canonical.matchAll(/^en ([0-9a-f]{32}) /gm)].map((match) => match[1]);
  assert.equal(new Set(ids).size, ids.length);
});

test("lifts ordered const bindings as lexical local semantics", () => {
  const localSource = `/**
 * @param {string} left
 * @param {string} right
 * @returns {string}
 */
export function Join(left, right) {
  const middle = left + "λ";
  const complete = middle + right;
  return complete;
}`;
  const canonical = liftJavaScript({ source: localSource, moduleG1: moduleV15G1, packagePath: "example.test/local-text", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090d0/);
  assert.match(canonical, /000000000000000000000000000090d1/);
  assert.match(canonical, /000000000000000000000000000090d2/);
});

test("lifts a closed native JavaScript function call graph", () => {
  const source = `
/** @param {string} value @returns {string} */
function decorate(value) { return "[" + value + "]"; }
/** @param {string} left @param {string} right @returns {string} */
function combine(left, right) { return decorate(left) + decorate(right); }
/** @param {string} left @param {string} right @returns {string} */
export function Render(left, right) { const joined = combine(left, right); return decorate(joined); }
`;
  const canonical = liftJavaScript({ source, moduleG1, packagePath: "example.test/calls", revision: 1 });
  assert.equal((canonical.match(/^en [0-9a-f]{32} 00000000000000000000000000009011 /gm) || []).length, 3);
  assert.equal((canonical.match(/^en [0-9a-f]{32} 00000000000000000000000000009060 /gm) || []).length, 4);
});

test("lifts structural objects as canonical records", () => {
  const source = `
/** @typedef {Object} Item
 * @property {string} Name
 * @property {boolean} Enabled
 */
/** @param {string} name @returns {string} */
export function Label(name) { const item = { Name: name, Enabled: true }; return item.Name; }
`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV16G1, packagePath: "example.test/records", revision: 1 });
  for (const recordSchema of ["9030", "9031", "9032", "9033"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${recordSchema}`));
});

test("lifts let assignment and while as mutable places", () => {
  const source = `/** @param {string} value @param {string} suffix @param {boolean} enabled @returns {string} */
export function AppendOnce(value, suffix, enabled) {
  let result = value;
  let remaining = enabled;
  while (remaining) { result = result + suffix; remaining = false; }
  return result;
}`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV18G1, packagePath: "example.test/mutation", revision: 1 });
  for (const mutationSchema of ["90e0", "90e1", "90e2", "90e3", "90e4"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${mutationSchema}`));
});

test("lifts bigint mutation and one-sided choice", () => {
  const source = `/** @param {bigint} original @param {bigint} replacement @param {boolean} enabled @returns {bigint} */
export function Choose(original, replacement, enabled) {
  let result = original;
  if (enabled) { result = replacement; }
  return result;
}`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV19G1, packagePath: "example.test/choice", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f0/);
  assert.match(canonical, /000000000000000000000000000090e3/);
});

test("lifts console observation as a Foundation effect invocation", () => {
  const source = `/** @param {boolean} first @param {boolean} second @returns {boolean} */
export function Observe(first, second) { console.log(first); console.log(second); return second; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV20G1, packagePath: "example.test/effect", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f1/);
  assert.match(canonical, /00000000000000000000000000000015/);
  assert.match(canonical, /00000000000000000000000000000016/);
});

test("lifts a fixed array construction and indexed read", () => {
  const source = `/** @param {bigint} first @param {bigint} second @param {bigint} third @param {bigint} index @returns {bigint} */
export function Pick(first, second, third, index) { return [first, second, third][index]; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f2/);
  assert.match(canonical, /000000000000000000000000000090f3/);
  assert.match(canonical, /000000000000000000000000000090f4/);
});

test("lifts a fixed i64 array parameter", () => {
  const source = `/** @param {bigint[3]} values @param {bigint} index @returns {bigint} */
export function Pick(values, index) { return values[index]; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array-boundary", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f2/);
  assert.match(canonical, /000000000000000000000000000090f4/);
});

test("lifts reduce as typed deterministic fold bindings", () => {
  const source = `/** @param {bigint[3]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => total + value, 0n); }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV22G1, packagePath: "example.test/fold", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f5/);
  assert.match(canonical, /000000000000000000000000000090f6/);
  assert.match(canonical, /000000000000000000000000000090f7/);
});
