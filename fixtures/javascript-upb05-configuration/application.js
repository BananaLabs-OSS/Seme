import { ConfigInput, InitializeConfig } from "./configuration.js";
import { InitializePolicy } from "./policy.js";

/** @typedef {Object} Runtime
 * @property {boolean} Ready
 * @property {bigint} Total
 * @property {string} Namespace
 */
export class Runtime { constructor(Ready, Total, Namespace) { this.Ready = Ready; this.Total = Total; this.Namespace = Namespace; } }

/** @param {Settings} settings @param {Policy} policy @returns {Seme.Result<Runtime,string>} */
export function Assemble(settings, policy) { return Seme.ok(new Runtime(settings.Enabled, BigInt.asIntN(64, settings.Limit + policy.Limit), settings.Namespace)); }

/** @param {boolean} enabled @param {bigint} limit @param {string} namespace @returns {bigint} */
export function Run(enabled, limit, namespace) {
  return BigInt.asIntN(64, limit + limit);
}
