const wrap = (value) => BigInt.asIntN(64, value);

function next(random) {
  let value = random.value;
  value ^= value << 13n; value ^= value >> 7n; value ^= value << 17n;
  random.value = BigInt.asUintN(64, value);
  return BigInt.asIntN(64, random.value);
}

export function initialState(sequence) {
  const boundary = sequence % 11 === 0 ? 9223372036854775807n : BigInt(sequence % 17);
  const values = sequence % 13 === 0 ? [1n, -1n, 2n] : [boundary, BigInt(sequence % 5), 2n];
  return { Name: `sequence-${sequence}`, Values: Object.freeze(values), Counters: new Map([[1n, boundary], [2n, 0n]]) };
}

export function generatedSequences() {
  const random = { value: 0x5eed1234abcdef01n };
  return Array.from({ length: 128 }, (_, sequence) => ({
    name: `sequence-${String(sequence).padStart(3, "0")}`,
    initial: initialState(sequence),
    commands: Array.from({ length: 16 }, (_, step) => {
      const sample = next(random);
      return {
        Key: step % 7 === 1 ? 99n : (step % 2 === 0 ? 1n : 2n),
        Index: step % 7 === 2 ? 999n : BigInt(Math.abs(Number(sample % 2n))),
        Delta: step % 8 === 3 ? 9223372036854775807n : wrap(sample),
        Amount: step % 8 === 4 ? -9223372036854775808n : wrap(sample >> 3n),
        Scale: step % 2 === 0,
      };
    }),
  }));
}

export function canonicalValue(value) {
  if (typeof value === "bigint") return value.toString();
  if (value instanceof Map) return [...value.entries()].sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0).map(([key, item]) => [key.toString(), canonicalValue(item)]);
  if (Array.isArray(value)) return value.map(canonicalValue);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([key, item]) => [key, canonicalValue(item)]));
  return value;
}
