/** @typedef {Object} ConfigInput
 * @property {boolean} Enabled
 * @property {bigint} Limit
 * @property {string} Namespace
 */
export class ConfigInput { constructor(Enabled, Limit, Namespace) { this.Enabled = Enabled; this.Limit = Limit; this.Namespace = Namespace; } }

/** @typedef {Object} Settings
 * @property {boolean} Enabled
 * @property {bigint} Limit
 * @property {string} Namespace
 */
export class Settings { constructor(Enabled, Limit, Namespace) { this.Enabled = Enabled; this.Limit = Limit; this.Namespace = Namespace; } }

/** @returns {boolean} */ export function DefaultEnabled() { return true; }
/** @returns {bigint} */ export function DefaultLimit() { return 8n; }
/** @returns {string} */ export function DefaultNamespace() { return "seme"; }
/** @param {bigint} value @returns {Seme.Result<bigint,string>} */ export function ValidateLimit(value) { return Seme.ok(value); }
/** @param {ConfigInput} input @returns {Seme.Result<Settings,string>} */
export function InitializeConfig(input) { return Seme.ok(new Settings(input.Enabled, input.Limit, input.Namespace)); }
