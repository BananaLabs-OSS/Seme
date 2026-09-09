/** @param {bigint} value @returns {Seme.Result<bigint,bigint>} */
function CheckPositive(value) {
  if (value <= 0n) {
    return Seme.error(99n);
  } else {
    return Seme.ok(value);
  }
}

/** @param {bigint} value @returns {Seme.Result<bigint,bigint>} */
export function IncrementPositive(value) {
  return Seme.matchResult(
    CheckPositive(value),
    (accepted) => Seme.ok(BigInt.asIntN(64, accepted + 1n)),
    (error) => Seme.error(error),
  );
}
