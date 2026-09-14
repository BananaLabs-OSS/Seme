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
const moduleV30G1 = fs.readFileSync(new URL("../../modules/execution/v30/module.g1", import.meta.url), "utf8");
const moduleV32G1 = fs.readFileSync(new URL("../../modules/execution/v32/module.g1", import.meta.url), "utf8");
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

test("projects Go-owned map range through an explicit JavaScript adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v70/program.g1", import.meta.url), "utf8");
  const projected = projectJavaScript(canonical);
  assert.match(projected, /Seme\.nativeRange\("go"/);
  assert.match(projected, /value/);
});

test("projects Go-owned slicing through an explicit JavaScript adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v71/program.g1", import.meta.url), "utf8");
  assert.match(projectJavaScript(canonical), /Seme\.nativeSlice\("go"/);
});

test("projects Go-owned pointer dereference through an explicit JavaScript adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v72/program.g1", import.meta.url), "utf8");
  assert.match(projectJavaScript(canonical), /Seme\.nativeDereference\("go"/);
});

test("projects Go-owned binary operators through explicit JavaScript adapters", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v73/program.g1", import.meta.url), "utf8");
  const projected = projectJavaScript(canonical);
  assert.match(projected, /Seme\.nativeBinary\("go", "<<"/);
  assert.match(projected, /Seme\.nativeBinary\("go", "\|"/);
});

test("projects Go-owned indexed assignment through an explicit JavaScript adapter", () => {
  const canonical = fs.readFileSync(new URL("../../fixtures/go-execution-v74/program.g1", import.meta.url), "utf8");
  assert.match(projectJavaScript(canonical), /Seme\.assignNativeIndex\("go"/);
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
export function Pick(first, second, third, index) { return Seme.index(Seme.array([first, second, third]), index); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /Seme\.index\(Seme\.array\(\[first, second, third\]\), index\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts a fixed array parameter", () => {
  const source = `/** @param {bigint[3]} values @param {bigint} index @returns {bigint} */
export function Pick(values, index) { return Seme.index(values, index); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array-boundary", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@param \{bigint\[3\]\} values/);
  assert.match(projected, /return Seme\.index\(values, index\);/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV21G1, packagePath: "example.test/fixed-array-boundary", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts deterministic folds", () => {
  const source = `/** @param {bigint[3]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV22G1, packagePath: "example.test/fold", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /values\.reduce\(\(total, value\) => BigInt\.asIntN\(64, \(total \+ value\)\), 0n\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV22G1, packagePath: "example.test/fold", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts runtime-sized slice folds", () => {
  const source = `/** @param {bigint[]} values @returns {bigint} */
export function Sum(values) { return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV23G1, packagePath: "example.test/slice", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /@param \{bigint\[\]\} values/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV23G1, packagePath: "example.test/slice", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts collection queries", () => {
  const source = `/** @param {bigint[]} values @param {bigint} fallback @returns {bigint} */
export function LastOr(values, fallback) { if (values.length <= 0n) return fallback; return Seme.index(values, values.length - 1); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV24G1, packagePath: "example.test/query", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /Seme\.length\(values\)/);
  assert.match(projected, /Seme\.index\(values, \(Seme\.length\(values\) - 1n\)\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV24G1, packagePath: "example.test/query", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts immutable collection results", () => {
  const source = `/** @param {bigint[]} values @param {bigint} index @param {bigint} replacement @param {bigint} appended @returns {bigint[]} */
export function UpdateAndAppend(values, index, replacement, appended) { return values.with(Number(index), replacement).concat([appended]); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV25G1, packagePath: "example.test/update", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /Seme\.append\(Seme\.update\(values, index, replacement\), appended\)/);
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
  add(delta) { const state = new Counter(BigInt.asIntN(64, this.value + delta)); return { state, result: state.value }; }
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
  Adjust(value) { return BigInt.asIntN(64, value + this.Offset); }
}
/** @typedef {Object} ScaleAdjuster
 * @property {bigint} Factor
 */
class ScaleAdjuster {
  constructor(Factor) { this.Factor = Factor; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return BigInt.asIntN(64, value * this.Factor); }
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
  Adjust(value) { return BigInt.asIntN(64, value + this.Offset); }
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
function MakeAdder(base) { return (value) => BigInt.asIntN(64, base + value); }
/** @param {function(bigint): bigint} fn @param {bigint} value @returns {bigint} */
function Apply(fn, value) { return fn(value); }
/** @param {bigint} base @param {bigint} value @returns {bigint} */
export function Run(base, value) { return Apply(MakeAdder(base), value); }`;
  const first = liftJavaScript({ source, moduleG1: moduleV28G1, packagePath: "example.test/immutable-closure", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /\(value\) => BigInt\.asIntN\(64, \(base \+ value\)\)/);
  assert.match(projected, /return fn\(value\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV28G1, packagePath: "example.test/immutable-closure", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts idiomatic mutable closures", () => {
  const source = `/** @param {bigint} start @returns {function(bigint): bigint} */
function MakeCounter(start) { let value = start; return (delta) => { value = BigInt.asIntN(64, value + delta); return value; }; }
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

test("projects and re-lifts native runtime-keyed Map folds", () => {
  const source = `/** @param {bigint[]} values @param {bigint} key @returns {bigint} */
export function Tally(values, key) { return values.reduce((counts, value) => new Map(counts).set(value, BigInt.asIntN(64, (counts.get(value) ?? 0n) + 1n)), new Map()).get(key) ?? 0n; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/runtime-map", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /Seme\.mapInsert\(counts/);
  assert.match(projected, /Seme\.mapLookupZero/);
  assert.match(projected, /Seme\.emptyMap\(\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV30G1, packagePath: "example.test/runtime-map", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts the cumulative state-flow proof", () => {
  const source = `/** @typedef {Object} Accumulator
 * @property {bigint} Value
 */
class Accumulator {
  constructor(Value) { this.Value = Value; }
  /** @param {bigint} delta @returns {Transition<Accumulator,bigint>} */
  Add(delta) { const next = new Accumulator(BigInt.asIntN(64, this.Value + delta)); return { state: next, result: next.Value }; }
}
/** @param {bigint[]} values @returns {bigint} */
function Sum(values) { return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n); }
/** @param {Accumulator} state @param {bigint[]} values @param {boolean} enabled @returns {Transition<Accumulator,bigint>} */
export function Run(state, values, enabled) { const delta = Sum(values); if (enabled) { return state.Add(delta); } else { return state.Add(0n); } }`;
  const first = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-state-flow", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /values\.reduce/);
  assert.match(projected, /if \(enabled\)/);
  assert.match(projected, /state\.Add\(delta\)/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-state-flow", revision: 1 });
  assert.equal(second, first);
});

test("projects and re-lifts the cumulative text-collection proof", () => {
  const source = `/** @param {bigint[]} values @returns {bigint} */
function Sum(values) { return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n); }
/** @param {string} prefix @param {bigint[]} values @returns {string} */
export function Describe(prefix, values) { const total = Sum(values); if (total <= 0n) { return prefix + ":non-positive"; } return prefix + ":positive"; }`;
  const first = liftJavaScript({ source, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-text-collection", revision: 1 });
  const projected = projectJavaScript(first);
  assert.match(projected, /values\.reduce/);
  assert.match(projected, /total <= 0n/);
  assert.match(projected, /prefix \+ ":non-positive"/);
  const second = liftJavaScript({ source: projected, moduleG1: moduleV30G1, packagePath: "example.test/cumulative-text-collection", revision: 1 });
  assert.equal(second, first);
});

test("projects bytes, Option, and Result constructors and re-lifts identically", () => {
  const sources = [
    "/** @returns {Uint8Array} */ export function Value() { return Seme.bytes([0, 255, 42]); }",
    "/** @returns {Seme.Option<bigint>} */ export function Value() { return Seme.none(); }",
    "/** @param {bigint} value @returns {Seme.Option<bigint>} */ export function Value(value) { return Seme.some(value); }",
    "/** @param {Uint8Array} value @returns {Seme.Result<Uint8Array,string>} */ export function Value(value) { return Seme.ok(value); }",
    "/** @param {string} error @returns {Seme.Result<Uint8Array,string>} */ export function Value(error) { return Seme.error(error); }",
  ];
  for (const [index, source] of sources.entries()) {
    const options = { moduleG1: moduleV32G1, packagePath: `example.test/project-composite/${index}`, revision: 1 };
    const canonical = liftJavaScript({ ...options, source });
    assert.equal(liftJavaScript({ ...options, source: projectJavaScript(canonical) }), canonical);
  }
});

test("projects total nested Option and Result matches and re-lifts identically", () => {
  const source = "/** @param {Seme.Option<Seme.Result<Uint8Array,string>>} value @returns {boolean} */ export function Accepted(value) { return Seme.matchOption(value, () => false, (result) => Seme.matchResult(result, (payload) => Seme.bytesEqual(payload, Seme.bytes([111, 107])), (error) => error === \"bad\")); }";
  const options = { moduleG1: moduleV32G1, packagePath: "example.test/project-composite-match", revision: 1 };
  const canonical = liftJavaScript({ ...options, source });
  assert.equal(liftJavaScript({ ...options, source: projectJavaScript(canonical) }), canonical);
});
