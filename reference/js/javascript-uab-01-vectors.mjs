const cases = [
  ["zero", "0", "0"],
  ["positive", "7", "5"],
  ["mixed-sign", "-7", "5"],
  ["overflow", "9223372036854775807", "1"],
  ["underflow", "-9223372036854775808", "-1"],
];

const valid = cases.map(([name, left, right]) => ({
  name,
  arguments: [{ kind: "i64", i64: left }, { kind: "i64", i64: right }],
  result: { kind: "i64", i64: BigInt.asIntN(64, BigInt(left) + BigInt(right)).toString() },
}));
const malformed = [
  { name: "wrong-arity", arguments: [{ kind: "i64", i64: "1" }] },
  { name: "wrong-type", arguments: [{ kind: "i64", i64: "1" }, { kind: "bool", bool: true }] },
];
process.stdout.write(`${JSON.stringify({ valid, malformed })}\n`);
