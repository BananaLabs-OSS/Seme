/** @typedef {Object} V1
 * @property {bigint} Count
 */
export class V1 { constructor(Count) { this.Count = Count; } }

/** @typedef {Object} V2
 * @property {bigint} Count
 * @property {string} Namespace
 */
export class V2 { constructor(Count, Namespace) { this.Count = Count; this.Namespace = Namespace; } }

/** @param {V1} value @returns {Seme.Result<V1,bigint>} */
export function ValidateV1(value) { return Seme.ok(value); }
/** @param {V2} value @returns {Seme.Result<V2,bigint>} */
export function ValidateV2(value) { return Seme.ok(value); }
/** @param {V1} value @returns {Seme.Result<V2,bigint>} */
export function MigrateV1ToV2(value) { return Seme.ok(new V2(value.Count, "migrated")); }
