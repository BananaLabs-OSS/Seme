/**
 * @param {bigint} value
 * @param {boolean} enabled
 * @param {string} label
 * @returns {string}
 */
export function Classify(value, enabled, label) {
  if (enabled && value <= 0n) return label + ":non-positive";
  return label + ":positive";
}
