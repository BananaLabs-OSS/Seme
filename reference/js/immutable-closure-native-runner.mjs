function MakeAdder(base) {
  return (value) => BigInt.asIntN(64, base + value);
}

function Apply(fn, value) {
  return fn(value);
}

function Run(base, value) {
  return Apply(MakeAdder(base), value);
}

if (Run(12n, 30n) !== 42n || Run(-50n, 8n) !== -42n) throw new Error("immutable closure result mismatch");
const left = MakeAdder(10n);
const right = MakeAdder(-10n);
if (left(1n) !== 11n || right(1n) !== -9n) throw new Error("closure environments aliased");
console.log(JSON.stringify({ positive: "42", negative: "-42", environments: "independent" }));
