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
export function Transform(values, index, replacement, appended, removeIndex, keepKey, removeKey) {
  const changed = Seme.remove(Seme.append(Seme.update(Seme.slice([Seme.index(values, 0n), Seme.index(values, 1n)]), index, replacement), appended), removeIndex);
  const total = changed.reduce((sum, value) => BigInt.asIntN(64, sum + value), 0n);
  const counters = Seme.mapRemove(Seme.mapInsert(Seme.mapInsert(Seme.emptyMap(), keepKey, total), removeKey, Seme.index(changed, index)), removeKey);
  return BigInt.asIntN(64, Seme.length(changed) + Seme.index(changed, index) + Seme.mapLookupZero(counters, keepKey));
}
