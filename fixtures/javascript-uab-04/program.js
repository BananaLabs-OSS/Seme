/** @param {bigint[]} values @param {bigint} index @param {bigint} replacement @param {bigint} appended @param {bigint} removeIndex @returns {bigint[]} */
/**
 * @param {bigint[]} values
 * @param {bigint} index
 * @param {bigint} replacement
 * @param {bigint} appended
 * @param {bigint} removeIndex
 * @param {bigint} keepKey
 * @param {bigint} removeKey
 * @returns {bigint}
 */
export function Evaluate(values, index, replacement, appended, removeIndex, keepKey, removeKey) {
  let total = Seme.remove(Seme.append(Seme.update(Seme.slice([Seme.index(values, 0n), Seme.index(values, 1n)]), index, replacement), appended), removeIndex).reduce((sum, value) => BigInt.asIntN(64, sum + value), 0n);
  return BigInt.asIntN(64,
    Seme.length(Seme.remove(Seme.append(Seme.update(Seme.slice([Seme.index(values, 0n), Seme.index(values, 1n)]), index, replacement), appended), removeIndex))
    + Seme.index(Seme.remove(Seme.append(Seme.update(Seme.slice([Seme.index(values, 0n), Seme.index(values, 1n)]), index, replacement), appended), removeIndex), index)
    + Seme.mapLookupZero(Seme.mapRemove(Seme.mapInsert(Seme.mapInsert(Seme.emptyMap(), keepKey, total), removeKey, Seme.index(Seme.remove(Seme.append(Seme.update(Seme.slice([Seme.index(values, 0n), Seme.index(values, 1n)]), index, replacement), appended), removeIndex), index)), removeKey), keepKey));
}
