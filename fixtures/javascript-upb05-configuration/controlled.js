/** @typedef {Object} ClockSample
 * @property {bigint} UnixMilliseconds
 * @property {bigint} Sequence
 */
export class ClockSample { constructor(UnixMilliseconds, Sequence) { this.UnixMilliseconds = UnixMilliseconds; this.Sequence = Sequence; } }

/** @param {ClockSample} sample @returns {bigint} */
export function ObserveClock(sample) { return sample.UnixMilliseconds; }

/** @typedef {Object} RandomState
 * @property {bigint} Value
 */
export class RandomState { constructor(Value) { this.Value = Value; } }

/** @typedef {Object} Draw
 * @property {bigint} State
 * @property {bigint} Value
 */
export class Draw { constructor(State, Value) { this.State = State; this.Value = Value; } }

/** @param {RandomState} state @returns {Draw} */
export function Next(state) {
  const value = BigInt.asIntN(64, state.Value * 48271n + 1n);
  return new Draw(value, value);
}

/** @typedef {Object} ControlledCommand
 * @property {bigint} Delta
 */
export class ControlledCommand { constructor(Delta) { this.Delta = Delta; } }

/** @typedef {Object} ControlledState
 * @property {bigint} Value
 */
export class ControlledState { constructor(Value) { this.Value = Value; } }

/** @typedef {Object} ControlledResult
 * @property {bigint} Value
 */
export class ControlledResult { constructor(Value) { this.Value = Value; } }

/** @param {ControlledState} state @param {ControlledCommand} command @returns {ControlledResult} */
export function DispatchControlled(state, command) {
  console.log(true);
  return new ControlledResult(BigInt.asIntN(64, state.Value + command.Delta));
}

/** @param {ControlledState} state @param {ControlledCommand} command @returns {ControlledResult} */
export function ReplayControlled(state, command) { return new ControlledResult(BigInt.asIntN(64, state.Value + command.Delta)); }
