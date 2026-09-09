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
function Adjust(adjuster, value) { return adjuster.Adjust(value); }

/** @param {bigint} base @returns {function(bigint): bigint} */
function MakeAdder(base) {
  return (value) => BigInt.asIntN(64, base + value);
}

/** @param {bigint} start @returns {function(bigint): bigint} */
function MakeCounter(start) {
  let total = start;
  return (delta) => {
    total = BigInt.asIntN(64, total + delta);
    return total;
  };
}

/**
 * @param {bigint[]} values
 * @param {boolean} scale
 * @param {bigint} delta
 * @param {bigint} amount
 * @returns {Seme.Result<bigint,bigint>}
 */
export function EvaluatePolicy(values, scale, delta, amount) {
  let position = 0n;
  const accumulate = MakeCounter(0n);
  let total = 0n;
  while (position <= BigInt.asIntN(64, Seme.length(values) - 1n)) {
    const value = Seme.index(values, position);
    if (value <= BigInt.asIntN(64, 0n - 1n)) return Seme.error(3n);
    total = accumulate(value);
    position = BigInt.asIntN(64, position + 1n);
  }
  const addAmount = MakeAdder(amount);
  if (scale) return Seme.ok(addAmount(Adjust(new ScaleAdjuster(delta), total)));
  return Seme.ok(addAmount(Adjust(new OffsetAdjuster(delta), total)));
}
