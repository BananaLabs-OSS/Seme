import assert from "node:assert/strict";
import test from "node:test";
import { ABI_V2_LIMITS, concatText, decodeText, encodeText, equalText, packTextFields, unpackRequest, unpackTextFields } from "./string-abi-v2.mjs";

test("exact UTF-8 vectors round trip", () => {
  const values = ["", "plain", "雪", "🦀", "a\0b", "é", "e\u0301"];
  assert.deepEqual(unpackTextFields(packTextFields(values), values.length), values);
  assert.equal(equalText("é", "e\u0301"), false);
  assert.equal(concatText("雪", "🦀"), "雪🦀");
});

test("source encoding rejects lone UTF-16 surrogates", () => {
  for (const value of ["\ud800", "\udc00", "x\ud800y"]) {
    assert.throws(() => encodeText(value), /invalid_source_text/);
  }
  assert.doesNotThrow(() => encodeText("\ud83e\udd80"));
});

test("fatal decoder rejects every malformed UTF-8 family", () => {
  const malformed = [[0xc0, 0x80], [0xe2, 0x82], [0x80], [0xed, 0xa0, 0x80], [0xf4, 0x90, 0x80, 0x80], [0xf5, 0x80, 0x80, 0x80]];
  for (const bytes of malformed) assert.throws(() => decodeText(Uint8Array.from(bytes)), /invalid_utf8/);
});

test("descriptor layout rejects gaps overlap truncation and trailing bytes", () => {
  const valid = packTextFields(["a", "bc"]);
  for (const mutate of [
    (view) => view.setUint32(0, 17, true),
    (view) => view.setUint32(8, 16, true),
    (view) => view.setUint32(12, 99, true),
  ]) {
    const changed = valid.slice(); mutate(new DataView(changed.buffer));
    assert.throws(() => unpackTextFields(changed, 2), /invalid_descriptor/);
  }
  assert.throws(() => unpackTextFields(valid.subarray(0, valid.length - 1), 2), /invalid_descriptor/);
  assert.throws(() => unpackTextFields(Uint8Array.from([...valid, 0]), 2), /trailing_bytes/);
});

test("bounds are exact and concatenation checks before allocation", () => {
  const maximum = "a".repeat(ABI_V2_LIMITS.textBytes);
  assert.equal(encodeText(maximum).length, ABI_V2_LIMITS.textBytes);
  assert.throws(() => encodeText(`${maximum}a`), /text_too_large/);
  assert.equal(concatText(maximum, ""), maximum);
  assert.throws(() => concatText(maximum, "a"), /concat_too_large/);
});

test("mixed scalar and text headers retain canonical parameter order", () => {
  const message = new Uint8Array(20);
  const view = new DataView(message.buffer);
  message[0] = 1;
  view.setBigInt64(1, -7n, true);
  view.setUint32(9, 17, true);
  view.setUint32(13, 3, true);
  message.set(encodeText("雪"), 17);
  assert.deepEqual(unpackRequest(message, ["bool", "i64", "string"]), [true, -7n, "雪"]);
  const noncanonical = message.slice(); noncanonical[0] = 2;
  assert.throws(() => unpackRequest(noncanonical, ["bool", "i64", "string"]), /noncanonical_boolean/);
});
