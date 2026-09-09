/**
 * @typedef {Object} Counter
 * @property {bigint} value
 */

/**
 * @param {Counter} counter
 * @param {bigint[2]} pair
 * @param {bigint[]} values
 * @param {Map<bigint,bigint>} counts
 * @param {bigint} key
 * @returns {bigint}
 */
export function Observe(counter, pair, values, counts, key) {
  return BigInt.asIntN(64, counter.value + Seme.index(pair, 1n) + Seme.length(values) + (counts.get(key) ?? 0n));
}
