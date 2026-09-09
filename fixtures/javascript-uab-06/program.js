/** @param {bigint} base @returns {function(bigint): bigint} */
function MakeAdder(base) {
  return (value) => BigInt.asIntN(64, base + value);
}

/** @param {bigint} base @param {bigint} value @returns {bigint} */
export function Immutable(base, value) {
  return MakeAdder(base)(value);
}

/** @param {bigint} start @returns {function(bigint): bigint} */
function MakeCounter(start) {
  let total = start;
  return (delta) => {
    total = BigInt.asIntN(64, total + delta);
    return total;
  };
}

/** @param {bigint} start @param {bigint} first @param {bigint} second @returns {bigint} */
export function Mutable(start, first, second) {
  const counter = MakeCounter(start);
  counter(first);
  return counter(second);
}
