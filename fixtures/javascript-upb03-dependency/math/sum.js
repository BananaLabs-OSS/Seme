/** @param {bigint} value @returns {bigint} */
function normalize(value) {
  return BigInt.asIntN(64, value);
}

/** @param {bigint} left @param {bigint} right @returns {bigint} */
export function Sum(left, right) {
  return normalize(BigInt.asIntN(64, left + right));
}
