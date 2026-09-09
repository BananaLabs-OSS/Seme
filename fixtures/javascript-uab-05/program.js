/** @interface Adjuster
 * @method Adjust
 * @param {bigint} value
 * @returns {bigint}
 */
/** @typedef {Object} OffsetAdjuster
 * @property {bigint} Amount
 */
class OffsetAdjuster {
  constructor(Amount) { this.Amount = Amount; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return BigInt.asIntN(64, value + this.Amount); }
}
/** @typedef {Object} ScaleAdjuster
 * @property {bigint} Amount
 */
class ScaleAdjuster {
  constructor(Amount) { this.Amount = Amount; }
  /** @param {bigint} value @returns {bigint} */
  Adjust(value) { return BigInt.asIntN(64, value * this.Amount); }
}
/** @param {Adjuster} adjuster @param {bigint} value @returns {bigint} */
function Apply(adjuster, value) {
  return adjuster.Adjust(value);
}
/** @param {boolean} scale @param {bigint} amount @param {bigint} value @returns {bigint} */
export function Dispatch(scale, amount, value) {
  if (scale) return Apply(new ScaleAdjuster(amount), value);
  return Apply(new OffsetAdjuster(amount), value);
}
