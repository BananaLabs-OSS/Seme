import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript } from "./javascript-provider.mjs";
import { projectJavaScript } from "./javascript-projector.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v14/module.g1", import.meta.url), "utf8");
const moduleV15G1 = fs.readFileSync(new URL("../../modules/execution/v15/module.g1", import.meta.url), "utf8");
const moduleV16G1 = fs.readFileSync(new URL("../../modules/execution/v16/module.g1", import.meta.url), "utf8");
const source = `/**
 * @param {string} left
 * @param {string} right
 * @returns {string}
 */
export function Join(left, right) {
  if (left === "") return right;
  return left + "λ" + right;
}`;

test("projects canonical text control to native JavaScript and re-lifts", () => {
  const canonical = liftJavaScript({ source, moduleG1, packagePath: "example.test/project", revision: 1 });
  const projected = projectJavaScript(canonical);
  assert.match(projected, /export function Join\(left, right\)/);
  assert.match(projected, /left === ""/);
  const relifted = liftJavaScript({ source: projected, moduleG1, packagePath: "example.test/project", revision: 1 });
  assert.equal(relifted.split("\n").slice(1).join("\n"), canonical.split("\n").slice(1).join("\n"));
});

test("projection rejects a graph without one executable program", () => {
  assert.throws(() => projectJavaScript(moduleG1), /javascript_projection\.requires_one_program/);
});

test("projection rejects duplicate semantic identities", () => {
  const canonical = liftJavaScript({ source, moduleG1, packagePath: "example.test/duplicate", revision: 1 });
  const entity = canonical.slice(canonical.indexOf("\nen ") + 1);
  assert.throws(() => projectJavaScript(`${canonical}\n${entity}`), /javascript_projection\.duplicate_entity/);
});

test("projects and re-lifts ordered lexical const bindings", () => {
  const localSource = source.replace('if (left === "") return right;\n  return left + "λ" + right;', 'const middle = left + "λ";\n  const complete = middle + right;\n  return complete;');
  const canonical = liftJavaScript({ source: localSource, moduleG1: moduleV15G1, packagePath: "example.test/project-locals", revision: 1 });
  const projected = projectJavaScript(canonical);
  assert.match(projected, /const middle =/);
  assert.match(projected, /const complete =/);
  const relifted = liftJavaScript({ source: projected, moduleG1: moduleV15G1, packagePath: "example.test/project-locals", revision: 1 });
  assert.equal(relifted.split("\n").slice(1).join("\n"), canonical.split("\n").slice(1).join("\n"));
});

test("projects and re-lifts a native JavaScript call graph", () => {
  const source = `
/** @param {string} value @returns {string} */
function decorate(value) { return "[" + value + "]"; }
/** @param {string} left @param {string} right @returns {string} */
function combine(left, right) { return decorate(left) + decorate(right); }
/** @param {string} left @param {string} right @returns {string} */
export function Render(left, right) { const joined = combine(left, right); return decorate(joined); }
`;
  const first = liftJavaScript({ source, moduleG1, packagePath: "example.test/calls", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /function decorate\(value\)/);
  assert.match(projected, /export function Render\(left, right\)/);
  assert.match(projected, /combine\(left, right\)/);
  const second = liftJavaScript({ source: projected, moduleG1, packagePath: "example.test/calls", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts canonical records as structural objects", () => {
  const source = `
/** @typedef {Object} Item
 * @property {string} Name
 * @property {boolean} Enabled
 */
/** @param {string} name @returns {string} */
export function Label(name) { const item = { Name: name, Enabled: true }; return item.Name; }
`;
  const first = liftJavaScript({ source, moduleG1: moduleV16G1, packagePath: "example.test/records", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@typedef \{Object\} Item/);
  assert.match(projected, /const item = \{ Name: name, Enabled: true \}/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV16G1, packagePath: "example.test/records", revision: 1 });
  assert.equal(second, first);
});
