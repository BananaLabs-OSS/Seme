const mode = process.argv[2] || "native";
let seed = 0x5e1e03n;
const next = () => { seed = (seed * 6364136223846793005n + 1442695040888963407n) & ((1n << 64n) - 1n); return BigInt.asIntN(64, seed); };
const cases = [
  [9n, 4n, false, 1n, 0n], [-2n, 4n, false, 1n, 0n], [3n, 4n, true, 0n, 1n], [9n, 4n, true, 0n, 1n],
  [(1n << 63n) - 1n, (1n << 63n) - 1n, true, 0n, 1n], [-(1n << 63n), -(1n << 63n), false, 1n, 0n],
];
for (let index = 0; index < 100; index += 1) { const value=next(),limit=next(),enabled=(next()&1n)===1n; cases.push([value,limit,enabled,enabled?0n:1n,enabled?1n:0n]); }
const observe = (value, limit, enabled, andIndex, orIndex) => {
  const original = value;
  let result = value;
  let running = enabled && [0n][Number(andIndex)] <= limit;
  while (running) {
    if (result <= limit) result = BigInt.asIntN(64, result + 1n);
    running = false;
  }
  const accepted = enabled || [0n][Number(orIndex)] <= limit;
  if (accepted) return result;
  return original;
};
const valid = cases.map(([value, limit, enabled, andIndex, orIndex], index) => ({ name: `case-${index}`, arguments: [value.toString(), limit.toString(), enabled, andIndex.toString(), orIndex.toString()], result: observe(value, limit, enabled, andIndex, orIndex).toString() }));
if (mode === "native") console.log(JSON.stringify({ valid }));
else if (mode === "canonical") console.log(JSON.stringify({
  valid: valid.map((item) => ({ name: item.name, arguments: [{kind:"i64",i64:item.arguments[0]},{kind:"i64",i64:item.arguments[1]},{kind:"bool",bool:item.arguments[2]},{kind:"i64",i64:item.arguments[3]},{kind:"i64",i64:item.arguments[4]}], result:{kind:"i64",i64:item.result} })),
  malformed: [
    {name:"and-rhs-not-skipped",arguments:[{kind:"i64",i64:"9"},{kind:"i64",i64:"4"},{kind:"bool",bool:true},{kind:"i64",i64:"1"},{kind:"i64",i64:"0"}]},
    {name:"or-rhs-not-skipped",arguments:[{kind:"i64",i64:"9"},{kind:"i64",i64:"4"},{kind:"bool",bool:false},{kind:"i64",i64:"0"},{kind:"i64",i64:"1"}]}
  ]
}));
else throw new Error("javascript.uab03.vector_mode");
