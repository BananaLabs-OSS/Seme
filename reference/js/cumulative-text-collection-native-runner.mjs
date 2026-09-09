function Sum(values) {
  return values.reduce((total, value) => BigInt.asIntN(64, total + value), 0n);
}

function Describe(prefix, values) {
  const total = Sum(values);
  if (total <= 0n) return prefix + ":non-positive";
  return prefix + ":positive";
}

const vectors = [
  ["sum", [4n, -1n], "sum:positive"],
  ["zero", [], "zero:non-positive"],
  ["neg", [-5n, 2n], "neg:non-positive"],
  ["\u4e16\u754c\ud83d\ude80", [1n], "\u4e16\u754c\ud83d\ude80:positive"],
];
for (const [prefix, values, expected] of vectors) if (Describe(prefix, values) !== expected) throw new Error("text-collection result mismatch");
console.log(JSON.stringify({ positive: "sum:positive", empty: "zero:non-positive", negative: "neg:non-positive", unicode: "exact" }));
