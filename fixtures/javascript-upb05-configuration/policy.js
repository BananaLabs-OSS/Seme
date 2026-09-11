/** @typedef {Object} Policy
 * @property {bigint} Limit
 */
export class Policy { constructor(Limit) { this.Limit = Limit; } }

/** @param {Settings} settings @returns {Seme.Result<Policy,string>} */
export function InitializePolicy(settings) { return Seme.ok(new Policy(settings.Limit)); }
