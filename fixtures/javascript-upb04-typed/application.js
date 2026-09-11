import { Transform } from "./policy/collections.js";

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
export function Run(values, index, replacement, appended, removeIndex, keepKey, removeKey) {
  return Transform(values, index, replacement, appended, removeIndex, keepKey, removeKey);
}
