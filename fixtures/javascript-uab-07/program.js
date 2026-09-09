/** @typedef {Object} Counter
 * @property {bigint} Value
 */

/** @param {Counter} counter @param {bigint} delta @returns {Transition<Counter,bigint>} */
export function Step(counter, delta) {
  return {
    state: { Value: BigInt.asIntN(64, counter.Value + delta) },
    result: counter.Value,
  };
}
