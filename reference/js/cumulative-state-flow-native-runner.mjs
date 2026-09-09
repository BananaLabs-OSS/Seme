class Accumulator {
  constructor(Value) { this.Value = Value; }
  Add(delta) {
    const next = new Accumulator(BigInt.asIntN(64, this.Value + delta));
    return { state: next, result: next.Value };
  }
}

function Sum(values) {
  return values.reduce((sum, value) => BigInt.asIntN(64, sum + value), 0n);
}

function Run(state, values, enabled) {
  const delta = Sum(values);
  if (enabled) return state.Add(delta);
  return state.Add(0n);
}

const initial = new Accumulator(10n);
const enabled = Run(initial, [5n, -2n, 9n], true);
const disabled = Run(initial, [5n, -2n, 9n], false);
const empty = Run(new Accumulator(-7n), [], true);
if (enabled.state.Value !== 22n || enabled.result !== 22n || disabled.result !== 10n || empty.result !== -7n || initial.Value !== 10n) throw new Error("cumulative state-flow mismatch");
console.log(JSON.stringify({ enabled: "22", disabled: "10", empty: "-7", original: "unchanged" }));
