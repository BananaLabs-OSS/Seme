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
const moduleV23G1 = fs.readFileSync(new URL("../../modules/execution/v23/module.g1", import.meta.url), "utf8");
const moduleV24G1 = fs.readFileSync(new URL("../../modules/execution/v24/module.g1", import.meta.url), "utf8");
const moduleV25G1 = fs.readFileSync(new URL("../../modules/execution/v25/module.g1", import.meta.url), "utf8");
const moduleV26G1 = fs.readFileSync(new URL("../../modules/execution/v26/module.g1", import.meta.url), "utf8");
const moduleV27G1 = fs.readFileSync(new URL("../../modules/execution/v27/module.g1", import.meta.url), "utf8");
const moduleV28G1 = fs.readFileSync(new URL("../../modules/execution/v28/module.g1", import.meta.url), "utf8");
const moduleV29G1 = fs.readFileSync(new URL("../../modules/execution/v29/module.g1", import.meta.url), "utf8");
const moduleV30G1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
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

test("lifts a runtime-sized bigint slice fold", () => {
  const source = `/** @param {bigint[]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => total + value, 0n); }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV23G1, packagePath: "example.test/slice", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f8/);
  assert.match(canonical, /000000000000000000000000000090f7/);
});

test("lifts collection length and computed slice indexing", () => {
  const source = `/** @param {bigint[]} values @param {bigint} fallback @returns {bigint} */
export function LastOr(values, fallback) { if (values.length <= 0n) return fallback; return values[values.length - 1]; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV24G1, packagePath: "example.test/query", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090f9/);
  assert.match(canonical, /000000000000000000000000000090fa/);
});

test("does not treat general JavaScript Number values as canonical i64", () => {
  const source = `/** @returns {bigint} */ export function Wrong() { return 1; }`;
  assert.throws(() => liftJavaScript({ source, moduleG1: moduleV24G1, packagePath: "example.test/wrong-number", revision: 1 }), /javascript\.unsupported_expression/);
});

test("lifts immutable collection update followed by append", () => {
  const source = `/** @param {bigint[]} values @param {bigint} index @param {bigint} replacement @param {bigint} appended @returns {bigint[]} */
export function UpdateAndAppend(values, index, replacement, appended) { return values.with(Number(index), replacement).concat([appended]); }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV25G1, packagePath: "example.test/update", revision: 1 });
  assert.match(canonical, /000000000000000000000000000090fb/);
  assert.match(canonical, /000000000000000000000000000090fc/);
});

test("lifts native methods as explicit immutable state transitions", () => {
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
  const canonical = liftJavaScript({ source, moduleG1: moduleV26G1, packagePath: "example.test/method-transition", revision: 1 });
  for (const methodSchema of ["a000", "a001", "a002", "a003", "a004", "a005"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${methodSchema}`));
});

test("lifts explicit state and result projections", () => {
  const base = `/** @typedef {Object} Counter
 * @property {bigint} value
 */
class Counter {
  constructor(value) { this.value = value; }
  /** @param {bigint} delta @returns {Transition<Counter,bigint>} */
  add(delta) { const state = new Counter(this.value + delta); return { state, result: state.value }; }
}`;
  const stateSource = `${base}\n/** @param {Counter} counter @param {bigint} delta @returns {Counter} */\nexport function Next(counter, delta) { return counter.add(delta).state; }`;
  const resultSource = `${base}\n/** @param {Counter} counter @param {bigint} delta @returns {bigint} */\nexport function Value(counter, delta) { return counter.add(delta).result; }`;
  assert.match(liftJavaScript({ source: stateSource, moduleG1: moduleV26G1, packagePath: "example.test/transition-state", revision: 1 }), /0000000000000000000000000000a006/);
  assert.match(liftJavaScript({ source: resultSource, moduleG1: moduleV26G1, packagePath: "example.test/transition-result", revision: 1 }), /0000000000000000000000000000a007/);
});

test("lifts native structural interfaces with explicit dispatch witnesses", () => {
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
  const canonical = liftJavaScript({ source, moduleG1: moduleV27G1, packagePath: "example.test/interface-dispatch", revision: 1 });
  for (const suffix of ["a010", "a011", "a012", "a014"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
  assert.equal((canonical.match(/0000000000000000000000000000a012/g) || []).length, 2);
});

test("lifts a concrete structural value through an explicit interface witness", () => {
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
  const canonical = liftJavaScript({ source, moduleG1: moduleV27G1, packagePath: "example.test/interface-value", revision: 1 });
  assert.match(canonical, /0000000000000000000000000000a013/);
  assert.match(canonical, /0000000000000000000000000000a014/);
});

test("lifts immutable lexical closures and indirect calls", () => {
  const source = `/** @param {bigint} base @returns {function(bigint): bigint} */
function MakeAdder(base) { return (value) => base + value; }
/** @param {function(bigint): bigint} fn @param {bigint} value @returns {bigint} */
function Apply(fn, value) { return fn(value); }
/** @param {bigint} base @param {bigint} value @returns {bigint} */
export function Run(base, value) { return Apply(MakeAdder(base), value); }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV28G1, packagePath: "example.test/immutable-closure", revision: 1 });
  for (const suffix of ["a020", "a021", "a022", "a023", "a024"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
});

test("lifts mutable closure environments and sequenced calls", () => {
  const source = `/** @param {bigint} start @returns {function(bigint): bigint} */
function MakeCounter(start) { let value = start; return (delta) => { value = value + delta; return value; }; }
/** @param {bigint} start @param {bigint} first @param {bigint} second @returns {bigint} */
export function Run(start, first, second) { const counter = MakeCounter(start); counter(first); return counter(second); }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV29G1, packagePath: "example.test/mutable-closure", revision: 1 });
  for (const suffix of ["a030", "a031", "a032", "a033", "a034", "a035"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
});

test("lifts runtime-keyed maps as immutable canonical folds", () => {
  const source = `/** @param {bigint[]} values @param {bigint} key @returns {bigint} */
export function Tally(values, key) { return values.reduce((counts, value) => new Map(counts).set(value, (counts.get(value) ?? 0n) + 1n), new Map()).get(key) ?? 0n; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/runtime-map", revision: 1 });
  for (const suffix of ["a040", "a041", "a042", "a043"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
  assert.match(canonical, /000000000000000000000000000090f7/);
});

test("compositionally lifts the cumulative state-flow proof", () => {
  const source = `/** @typedef {Object} Accumulator
 * @property {bigint} Value
 */
class Accumulator {
  constructor(Value) { this.Value = Value; }
  /** @param {bigint} delta @returns {Transition<Accumulator,bigint>} */
  Add(delta) { const next = new Accumulator(this.Value + delta); return { state: next, result: next.Value }; }
}
/** @param {bigint[]} values @returns {bigint} */
function Sum(values) { return values.reduce((total, value) => total + value, 0n); }
/** @param {Accumulator} state @param {bigint[]} values @param {boolean} enabled @returns {Transition<Accumulator,bigint>} */
export function Run(state, values, enabled) { const delta = Sum(values); if (enabled) { return state.Add(delta); } else { return state.Add(0n); } }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-state-flow", revision: 1 });
  for (const suffix of ["9015", "90d0", "90d2", "90c0", "90f7", "a003", "a005"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
});

test("compositionally lifts the cumulative text-collection proof", () => {
  const source = `/** @param {bigint[]} values @returns {bigint} */
function Sum(values) { return values.reduce((total, value) => total + value, 0n); }
/** @param {string} prefix @param {bigint[]} values @returns {string} */
export function Describe(prefix, values) { const total = Sum(values); if (total <= 0n) { return prefix + ":non-positive"; } return prefix + ":positive"; }`;
  const canonical = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-text-collection", revision: 1 });
  for (const suffix of ["9060", "90d0", "90d2", "90c0", "90c3", "90f7"]) assert.match(canonical, new RegExp(`0000000000000000000000000000${suffix}`));
});
