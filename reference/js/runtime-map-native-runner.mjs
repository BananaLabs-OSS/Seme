function Tally(values, key) {
  const counts = values.reduce((state, value) => new Map(state).set(value, (state.get(value) ?? 0n) + 1n), new Map());
  return counts.get(key) ?? 0n;
}

const repeated = [3n, -7n, 3n, 3n, -7n];
if (Tally(repeated, 3n) !== 3n || Tally(repeated, -7n) !== 2n || Tally(repeated, 99n) !== 0n) throw new Error("map tally mismatch");
if (Tally([...repeated].reverse(), 3n) !== 3n || Tally([], 3n) !== 0n) throw new Error("map ordering or empty mismatch");
const bounded = Array.from({ length: 512 }, (_, index) => BigInt(index % 17));
if (Tally(bounded, 1n) !== 31n) throw new Error("bounded map tally mismatch");
console.log(JSON.stringify({ repeated: "3", negative: "2", missing: "0", permutation: "stable", maximum: 512 }));
