import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { liftJavaScript } from "./javascript-provider.mjs";
import { projectJavaScript } from "./javascript-projector.mjs";

const moduleG1 = fs.readFileSync(new URL("../../modules/execution/v14/module.g1", import.meta.url), "utf8");
const moduleV15G1 = fs.readFileSync(new URL("../../modules/execution/v15/module.g1", import.meta.url), "utf8");
const moduleV16G1 = fs.readFileSync(new URL("../../modules/execution/v16/module.g1", import.meta.url), "utf8");
const moduleV18G1 = fs.readFileSync(new URL("../../modules/execution/v18/module.g1", import.meta.url), "utf8");
const moduleV19G1 = fs.readFileSync(new URL("../../modules/execution/v19/module.g1", import.meta.url), "utf8");
const moduleV20G1 = fs.readFileSync(new URL("../../modules/execution/v20/module.g1", import.meta.url), "utf8");
const moduleV21G1 = fs.readFileSync(new URL("../../modules/execution/v21/module.g1", import.meta.url), "utf8");
const moduleV22G1 = fs.readFileSync(new URL("../../modules/execution/v22/module.g1", import.meta.url), "utf8");
const moduleV23G1 = fs.readFileSync(new URL("../../modules/execution/v23/module.g1", import.meta.url), "utf8");
const moduleV24G1 = fs.readFileSync(new URL("../../modules/execution/v24/module.g1", import.meta.url), "utf8");
const moduleV25G1 = fs.readFileSync(new URL("../../modules/execution/v25/module.g1", import.meta.url), "utf8");
const moduleV26G1 = fs.readFileSync(new URL("../../modules/execution/v26/module.g1", import.meta.url), "utf8");
const moduleV27G1 = fs.readFileSync(new URL("../../modules/execution/v27/module.g1", import.meta.url), "utf8");
const moduleV28G1 = fs.readFileSync(new URL("../../modules/execution/v28/module.g1", import.meta.url), "utf8");
const moduleV29G1 = fs.readFileSync(new URL("../../modules/execution/v29/module.g1", import.meta.url), "utf8");
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

test("projects and re-lifts mutable places and while", () => {
  const source = `/** @param {string} value @param {string} suffix @param {boolean} enabled @returns {string} */
export function AppendOnce(value, suffix, enabled) {
  let result = value;
  let remaining = enabled;
  while (remaining) { result = result + suffix; remaining = false; }
  return result;
}`;
  const first = liftJavaScript({ source, moduleG1: moduleV18G1, packagePath: "example.test/mutation", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /let result = value/);
  assert.match(projected, /while \(remaining\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV18G1, packagePath: "example.test/mutation", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts bigint mutation and one-sided choice", () => {
  const source = `/** @param {bigint} original @param {bigint} replacement @param {boolean} enabled @returns {bigint} */
export function Choose(original, replacement, enabled) {
  let result = original;
  if (enabled) { result = replacement; }
  return result;
}`;
  const first = liftJavaScript({ source, moduleG1: moduleV19G1, packagePath: "example.test/choice", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@param \{bigint\} original/);
  assert.match(projected, /if \(enabled\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV19G1, packagePath: "example.test/choice", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts ordered observation effects", () => {
  const source = `/** @param {boolean} first @param {boolean} second @returns {boolean} */
export function Observe(first, second) { console.log(first); console.log(second); return second; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV20G1, packagePath: "example.test/effect", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /console\.log\(first\);[\s\S]*console\.log\(second\);/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV20G1, packagePath: "example.test/effect", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts fixed array indexed reads", () => {
  const source = `/** @param {bigint} first @param {bigint} second @param {bigint} third @param {bigint} index @returns {bigint} */
export function Pick(first, second, third, index) { return [first, second, third][index]; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /\[first, second, third\]\[index\]/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts a fixed array parameter", () => {
  const source = `/** @param {bigint[3]} values @param {bigint} index @returns {bigint} */
export function Pick(values, index) { return values[index]; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array-boundary", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@param \{bigint\[3\]\} values/);
  assert.match(projected, /return values\[index\];/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array-boundary", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts deterministic folds", () => {
  const source = `/** @param {bigint[3]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => total + value, 0n); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV22G1, packagePath: "example.test/fold", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /values\.reduce\(\(total, value\) => \(total \+ value\), 0n\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV22G1, packagePath: "example.test/fold", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts runtime-sized slice folds", () => {
  const source = `/** @param {bigint[]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => total + value, 0n); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV23G1, packagePath: "example.test/slice", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@param \{bigint\[\]\} values/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV23G1, packagePath: "example.test/slice", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts collection queries", () => {
  const source = `/** @param {bigint[]} values @param {bigint} fallback @returns {bigint} */
export function LastOr(values, fallback) { if (values.length <= 0n) return fallback; return values[values.length - 1]; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV24G1, packagePath: "example.test/query", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /values\.length/);
  assert.match(projected, /values\[\(values\.length - 1\)\]/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV24G1, packagePath: "example.test/query", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts immutable collection results", () => {
  const source = `/** @param {bigint[]} values @param {bigint} index @param {bigint} replacement @param {bigint} appended @returns {bigint[]} */
export function UpdateAndAppend(values, index, replacement, appended) { return values.with(Number(index), replacement).concat([appended]); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV25G1, packagePath: "example.test/update", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /\.with\(Number\(index\), replacement\)\.concat\(\[appended\]\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV25G1, packagePath: "example.test/update", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts native JavaScript method transitions", () => {
  const source = `/** @typedef {Object} Counter
 * @property {bigint} value
 */
class Counter {
  constructor(value) { this.value = value; }
  /** @param {bigint} delta @returns {Transition<Counter,bigint>} */
  add(delta) { const state = new Counter(this.value + delta); return { state, result: state.value }; }
}
/** @param {Counter} counter @param {bigint} delta @returns {Transition<Counter,bigint>} */
export function Step(counter, delta) { return counter.add(delta); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV26G1, packagePath: "example.test/method-transition", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /class Counter/);
  assert.match(projected, /add\(delta\)/);
  assert.match(projected, /new Counter\(/);
  assert.match(projected, /counter\.add\(delta\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV26G1, packagePath: "example.test/method-transition", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts native JavaScript structural dispatch", () => {
  const source = `/** @interface Adjuster
 * @method Adjust
 * @param {bigint} value
 * @returns {bigint}
 */
/** @typedef {Object} OffsetAdjuster
 * @property {bigint} Offset
 */
class OffsetAdjuster {
  constructor(Offset) { this.Offset = Offset; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return value + this.Offset; }
}
/** @typedef {Object} ScaleAdjuster
 * @property {bigint} Factor
 */
class ScaleAdjuster {
  constructor(Factor) { this.Factor = Factor; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return value * this.Factor; }
}
/** @param {Adjuster} adjuster @param {bigint} value @returns {bigint} */
export function Apply(adjuster, value) { return adjuster.Adjust(value); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV27G1, packagePath: "example.test/interface-dispatch", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@interface Adjuster/);
  assert.match(projected, /class OffsetAdjuster/);
  assert.match(projected, /class ScaleAdjuster/);
  assert.match(projected, /adjuster\.Adjust\(value\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV27G1, packagePath: "example.test/interface-dispatch", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts concrete values dispatched through structural interfaces", () => {
  const source = `/** @interface Adjuster
 * @method Adjust
 * @param {bigint} value
 * @returns {bigint}
 */
/** @typedef {Object} OffsetAdjuster
 * @property {bigint} Offset
 */
class OffsetAdjuster {
  constructor(Offset) { this.Offset = Offset; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return value + this.Offset; }
}
/** @param {bigint} offset @param {bigint} value @returns {bigint} */
export function ApplyOffset(offset, value) { return new OffsetAdjuster(offset).Adjust(value); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV27G1, packagePath: "example.test/interface-value", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /new OffsetAdjuster\(offset\)\.Adjust\(value\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV27G1, packagePath: "example.test/interface-value", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts immutable lexical closures", () => {
  const source = `/** @param {bigint} base @returns {function(bigint): bigint} */
function MakeAdder(base) { return (value) => base + value; }
/** @param {function(bigint): bigint} fn @param {bigint} value @returns {bigint} */
function Apply(fn, value) { return fn(value); }
/** @param {bigint} base @param {bigint} value @returns {bigint} */
export function Run(base, value) { return Apply(MakeAdder(base), value); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV28G1, packagePath: "example.test/immutable-closure", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /\(value\) => \(base \+ value\)/);
  assert.match(projected, /return fn\(value\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV28G1, packagePath: "example.test/immutable-closure", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts idiomatic mutable closures", () => {
  const source = `/** @param {bigint} start @returns {function(bigint): bigint} */
function MakeCounter(start) { let value = start; return (delta) => { value = value + delta; return value; }; }
/** @param {bigint} start @param {bigint} first @param {bigint} second @returns {bigint} */
export function Run(start, first, second) { const counter = MakeCounter(start); counter(first); return counter(second); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV29G1, packagePath: "example.test/mutable-closure", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /let value = start/);
  assert.match(projected, /counter\(first\)/);
  assert.match(projected, /return counter\(second\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV29G1, packagePath: "example.test/mutable-closure", revision: 1 });
  assert.equal(second, first);
});
