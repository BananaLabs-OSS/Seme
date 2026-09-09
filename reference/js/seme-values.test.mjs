import assert from "node:assert/strict";
import test from "node:test";
import { Seme } from "./seme-values.mjs";

test("explicit native composite values preserve tags and exact bytes", () => {
  const array = Seme.array([-7n, 0n, 42n]);
  assert.equal(Seme.index(array, 2n), 42n);
  assert.equal(Seme.length(array), 3n);
  assert.throws(() => Seme.index(array, -1n), /seme.index_out_of_bounds/);
  assert.throws(() => Seme.index(array, 3n), /seme.index_out_of_bounds/);
  const octets = Seme.bytes([0, 255, 42]);
  assert.deepEqual([...octets], [0, 255, 42]);
  assert.deepEqual(Seme.none(), { tag: "none" });
  assert.deepEqual(Seme.some(7n), { tag: "some", value: 7n });
  assert.deepEqual(Seme.ok(octets), { tag: "ok", value: octets });
  assert.deepEqual(Seme.error("bad"), { tag: "error", value: "bad" });
  assert.throws(() => { Seme.some(7n).tag = "none"; }, TypeError);
  assert.throws(() => Seme.bytes([256]), /seme.invalid_bytes/);
  assert.equal(Seme.bytesEqual(Seme.bytes([0, 255]), Seme.bytes([0, 255])), true);
  assert.equal(Seme.bytesEqual(Seme.bytes([0]), Seme.bytes([1])), false);
  assert.equal(Seme.matchOption(Seme.some(7n), () => 0n, (value) => value), 7n);
  assert.equal(Seme.matchResult(Seme.error("bad"), () => true, () => false), false);
  assert.throws(() => Seme.matchOption({ tag: "unknown" }, () => 0, () => 1), /seme.invalid_option/);
});

test("collection adapters are immutable and keep raw JavaScript semantics separate", () => {
  const source = [1n, 2n];
  const slice = Seme.slice(source);
  const appended = Seme.append(slice, 3n);
  const updated = Seme.update(appended, 1n, 9n);
  const shortened = Seme.remove(updated, 0n);
  assert.deepEqual([...slice], [1n, 2n]);
  assert.deepEqual([...appended], [1n, 2n, 3n]);
  assert.deepEqual([...updated], [1n, 9n, 3n]);
  assert.deepEqual([...shortened], [9n, 3n]);
  assert.throws(() => Seme.update(slice, -1n, 0n), /seme.update_out_of_bounds/);
  assert.throws(() => Seme.remove(slice, 2n), /seme.remove_out_of_bounds/);
  const raw = [1n]; raw.push(2n); assert.deepEqual(raw, [1n, 2n]);
  const empty = Seme.emptyMap();
  const inserted = Seme.mapInsert(empty, 7n, 8n);
  const removed = Seme.mapRemove(inserted, 7n);
  assert.equal(Seme.mapLookupZero(empty, 7n), 0n);
  assert.equal(Seme.mapLookupZero(inserted, 7n), 8n);
  assert.equal(Seme.mapLookupZero(removed, 7n), 0n);
  assert.equal(empty.size, 0);
});
