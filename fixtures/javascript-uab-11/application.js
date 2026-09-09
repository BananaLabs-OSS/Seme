import { EvaluatePolicy } from "./policy.js";

/** @typedef {Object} State
 * @property {string} Name
 * @property {bigint[]} Values
 * @property {Map<bigint,bigint>} Counters
 */

/** @typedef {Object} Command
 * @property {bigint} Key
 * @property {bigint} Index
 * @property {bigint} Delta
 * @property {bigint} Amount
 * @property {boolean} Scale
 */

/**
 * @param {State} state
 * @param {Command} command
 * @returns {Seme.Result<Transition<State,bigint>,bigint>}
 */
export function Apply(state, command) {
  return Seme.matchOption(
    Seme.mapLookup(state.Counters, command.Key),
    () => Seme.error(1n),
    (counter) => {
      if (command.Index <= BigInt.asIntN(64, 0n - 1n) || Seme.length(state.Values) <= command.Index) return Seme.error(2n);
      return Seme.matchResult(
        EvaluatePolicy(state.Values, command.Scale, command.Delta, command.Amount),
        (policyValue) => {
          const nextCounter = BigInt.asIntN(64, counter + policyValue);
          const nextValues = Seme.append(Seme.update(state.Values, command.Index, policyValue), policyValue);
          const nextCounters = Seme.mapInsert(state.Counters, command.Key, nextCounter);
          console.log(true);
          return Seme.ok({ state: { Name: state.Name, Values: nextValues, Counters: nextCounters }, result: nextCounter });
        },
        (error) => Seme.error(error),
      );
    },
  );
}
