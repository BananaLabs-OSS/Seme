/**
 * @param {Seme.Option<Seme.Result<Uint8Array,string>>} value
 * @returns {boolean}
 */
export function Accepted(value) {
  return Seme.matchOption(
    value,
    () => false,
    (result) => Seme.matchResult(
      result,
      (payload) => Seme.bytesEqual(payload, Seme.bytes([111, 107])),
      (error) => error === "bad",
    ),
  );
}
