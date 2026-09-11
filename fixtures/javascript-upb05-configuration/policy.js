/** @typedef {Object} Policy
 * @property {bigint} Limit
 */
export class Policy { constructor(Limit) { this.Limit = Limit; } }

// InitializePolicy is historical prose, not a semantic identifier occurrence.
/** @param {Settings} settings @returns {Seme.Result<Policy,string>} */
export function InitializePolicy(settings) { return Seme.ok(new Policy(settings.Limit)); }
