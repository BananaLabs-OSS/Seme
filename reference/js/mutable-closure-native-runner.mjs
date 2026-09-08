function MakeCounter(start) {
  let value = start;
  return (delta) => {
    value = BigInt.asIntN(64, value + delta);
    return value;
  };
}

function Run(start, first, second) {
  const counter = MakeCounter(start);
  counter(first);
  return counter(second);
}

if (Run(10n, 5n, 7n) !== 22n || Run(-10n, -5n, 7n) !== -8n) throw new Error("mutable closure result mismatch");
const left = MakeCounter(10n);
const right = MakeCounter(-10n);
if (left(1n) !== 11n || left(1n) !== 12n || right(1n) !== -9n) throw new Error("mutable closure environments aliased");
console.log(JSON.stringify({ positive: "22", negative: "-8", repeated: "committed", environments: "independent" }));
