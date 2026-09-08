class Counter {
  constructor(value) {
    this.value = value;
  }

  add(delta) {
    const state = new Counter(this.value + delta);
    return { state, result: state.value };
  }
}

const original = new Counter(-7n);
const transition = original.add(12n);
if (transition.state.value !== 5n || transition.result !== 5n || original.value !== -7n) {
  throw new Error("native JavaScript transition mismatch");
}
console.log(JSON.stringify({ state: transition.state.value.toString(), result: transition.result.toString(), original: original.value.toString() }));
