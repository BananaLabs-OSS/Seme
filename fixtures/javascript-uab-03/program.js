/**
 * Bounded UAB-03 control-flow proof. `bigint` parameters are the explicit
 * native realization of canonical wrapping i64; Boolean inputs are refined
 * at the invocation boundary and never use general JavaScript truthiness.
 * @param {bigint} value
 * @param {bigint} limit
 * @param {boolean} enabled
 * @param {bigint} andIndex
 * @param {bigint} orIndex
 * @returns {bigint}
 */
export function Evaluate(value, limit, enabled, andIndex, orIndex) {
  let original = value;
  let result = value;
  let running = enabled && Seme.index(Seme.array([0n]), andIndex) <= limit;
  while (running) {
    if (result <= limit) {
      result = BigInt.asIntN(64, result + 1n);
    }
    running = false;
  }
  if (enabled || Seme.index(Seme.array([0n]), orIndex) <= limit) return result;
  return original;
}
