/** @typedef {Object} TransportCommand
 * @property {bigint} Sequence
 * @property {bigint} Value
 */
export class TransportCommand { constructor(Sequence, Value) { this.Sequence = Sequence; this.Value = Value; } }

/** @typedef {Object} TransportEvent
 * @property {bigint} Sequence
 * @property {bigint} Value
 */
export class TransportEvent { constructor(Sequence, Value) { this.Sequence = Sequence; this.Value = Value; } }

/** @param {TransportCommand} command @returns {TransportEvent} */
export function Dispatch(command) { return new TransportEvent(command.Sequence, BigInt.asIntN(64, command.Value + 1n)); }

/** @param {TransportEvent} event @returns {bigint} */
export function Replay(event) { return event.Value; }
