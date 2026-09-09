/** @param {bigint} left @param {bigint} right @returns {bigint} */
export function Sum(left, right) { return BigInt.asIntN(64, left + right); }
