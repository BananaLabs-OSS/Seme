const freeze = (value) => Object.freeze(value);

// Explicit native values for meanings JavaScript does not possess directly.
export const Seme = Object.freeze({
  array(values) {
    if (!Array.isArray(values)) throw new TypeError("seme.invalid_array");
    return Object.freeze([...values]);
  },
  index(values, index) {
    if (!Array.isArray(values) || typeof index !== "bigint" || index < 0n || index >= BigInt(values.length)) throw new RangeError("seme.index_out_of_bounds");
    return values[Number(index)];
  },
  length(values) {
    if (!Array.isArray(values)) throw new TypeError("seme.invalid_collection");
    return BigInt(values.length);
  },
  slice(values) {
    if (!Array.isArray(values)) throw new TypeError("seme.invalid_slice");
    return Object.freeze([...values]);
  },
  append(values, value) {
    if (!Array.isArray(values) || values.length >= 512) throw new RangeError("seme.invalid_append");
    return Object.freeze([...values, value]);
  },
  update(values, index, value) {
    if (!Array.isArray(values) || typeof index !== "bigint" || index < 0n || index >= BigInt(values.length)) throw new RangeError("seme.update_out_of_bounds");
    const result = [...values]; result[Number(index)] = value; return Object.freeze(result);
  },
  remove(values, index) {
    if (!Array.isArray(values) || typeof index !== "bigint" || index < 0n || index >= BigInt(values.length)) throw new RangeError("seme.remove_out_of_bounds");
    return Object.freeze([...values.slice(0, Number(index)), ...values.slice(Number(index) + 1)]);
  },
  emptyMap() { return new Map(); },
  mapLookupZero(values, key) {
    if (!(values instanceof Map)) throw new TypeError("seme.invalid_map");
    return values.get(key) ?? 0n;
  },
  mapInsert(values, key, value) {
    if (!(values instanceof Map) || (!values.has(key) && values.size >= 512)) throw new RangeError("seme.invalid_map_insert");
    return new Map(values).set(key, value);
  },
  mapRemove(values, key) {
    if (!(values instanceof Map)) throw new TypeError("seme.invalid_map");
    const result = new Map(values); result.delete(key); return result;
  },
  bytes(values) {
    if (!Array.isArray(values) || values.some((value) => !Number.isInteger(value) || value < 0 || value > 255)) throw new TypeError("seme.invalid_bytes");
    return Uint8Array.from(values);
  },
  bytesEqual(left, right) {
    return left instanceof Uint8Array && right instanceof Uint8Array && left.length === right.length && left.every((value, index) => value === right[index]);
  },
  none() { return freeze({ tag: "none" }); },
  some(value) { return freeze({ tag: "some", value }); },
  ok(value) { return freeze({ tag: "ok", value }); },
  error(value) { return freeze({ tag: "error", value }); },
  matchOption(value, none, some) {
    if (value?.tag === "none") return none();
    if (value?.tag === "some") return some(value.value);
    throw new TypeError("seme.invalid_option");
  },
  matchResult(value, ok, error) {
    if (value?.tag === "ok") return ok(value.value);
    if (value?.tag === "error") return error(value.value);
    throw new TypeError("seme.invalid_result");
  },
});
