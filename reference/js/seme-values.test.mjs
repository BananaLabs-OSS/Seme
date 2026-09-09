import assert from "node:assert/strict";
import test from "node:test";
import { Seme } from "./seme-values.mjs";

test("explicit native composite values preserve tags and exact bytes", () => {
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
