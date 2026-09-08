const strictUTF8 = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true });
const utf8 = new TextEncoder();

export const ABI_V2_LIMITS = Object.freeze({ textBytes: 4096, messageBytes: 7160 });

export function encodeText(value) {
  if (typeof value !== "string" || hasLoneSurrogate(value)) {
    throw new Error("string_abi_v2.invalid_source_text");
  }
  const encoded = utf8.encode(value);
  if (encoded.length > ABI_V2_LIMITS.textBytes) {
    throw new Error("string_abi_v2.text_too_large");
  }
  return encoded;
}

export function decodeText(encoded) {
  const bytes = asBytes(encoded);
  if (bytes.length > ABI_V2_LIMITS.textBytes) throw new Error("string_abi_v2.text_too_large");
  try {
    return strictUTF8.decode(bytes);
  } catch {
    throw new Error("string_abi_v2.invalid_utf8");
  }
}

export function packTextFields(values) {
  const fields = values.map(encodeText);
  const headerSize = checkedMultiply(fields.length, 8);
  const total = fields.reduce((size, field) => checkedAdd(size, field.length), headerSize);
  if (total > ABI_V2_LIMITS.messageBytes) throw new Error("string_abi_v2.message_too_large");
  const message = new Uint8Array(total);
  const view = new DataView(message.buffer);
  let offset = headerSize;
  for (let index = 0; index < fields.length; index += 1) {
    view.setUint32(index * 8, offset, true);
    view.setUint32(index * 8 + 4, fields[index].length, true);
    message.set(fields[index], offset);
    offset += fields[index].length;
  }
  return message;
}

export function unpackTextFields(encoded, fieldCount) {
  const message = asBytes(encoded);
  if (!Number.isSafeInteger(fieldCount) || fieldCount < 0) throw new Error("string_abi_v2.invalid_field_count");
  const headerSize = checkedMultiply(fieldCount, 8);
  if (message.length < headerSize || message.length > ABI_V2_LIMITS.messageBytes) {
    throw new Error("string_abi_v2.invalid_message_length");
  }
  const view = new DataView(message.buffer, message.byteOffset, message.byteLength);
  const values = [];
  let expectedOffset = headerSize;
  for (let index = 0; index < fieldCount; index += 1) {
    const offset = view.getUint32(index * 8, true);
    const length = view.getUint32(index * 8 + 4, true);
    if (offset !== expectedOffset || length > ABI_V2_LIMITS.textBytes || offset > message.length || length > message.length - offset) {
      throw new Error("string_abi_v2.invalid_descriptor");
    }
    const field = message.subarray(offset, offset + length);
    values.push(decodeText(field));
    expectedOffset = checkedAdd(offset, length);
  }
  if (expectedOffset !== message.length) throw new Error("string_abi_v2.trailing_bytes");
  return values;
}

export function unpackRequest(encoded, types) {
  const message = asBytes(encoded);
  const headerSize = requestHeaderSize(types);
  if (message.length < headerSize || message.length > ABI_V2_LIMITS.messageBytes) throw new Error("string_abi_v2.invalid_message_length");
  const view = new DataView(message.buffer, message.byteOffset, message.byteLength);
  const values = [];
  let header = 0;
  let payload = headerSize;
  for (const type of types) {
    if (type === "bool") {
      const value = message[header];
      if (value !== 0 && value !== 1) throw new Error("string_abi_v2.noncanonical_boolean");
      values.push(value === 1); header += 1;
    } else if (type === "i64") {
      values.push(view.getBigInt64(header, true)); header += 8;
    } else if (type === "string") {
      const offset = view.getUint32(header, true);
      const length = view.getUint32(header + 4, true);
      if (offset !== payload || length > ABI_V2_LIMITS.textBytes || offset > message.length || length > message.length - offset) throw new Error("string_abi_v2.invalid_descriptor");
      values.push(decodeText(message.subarray(offset, offset + length)));
      payload = checkedAdd(offset, length); header += 8;
    } else throw new Error("string_abi_v2.unsupported_type");
  }
  if (payload !== message.length) throw new Error("string_abi_v2.trailing_bytes");
  return values;
}

export function concatText(left, right) {
  const leftBytes = encodeText(left);
  const rightBytes = encodeText(right);
  if (leftBytes.length > ABI_V2_LIMITS.textBytes - rightBytes.length) {
    throw new Error("string_abi_v2.concat_too_large");
  }
  return decodeText(Uint8Array.from([...leftBytes, ...rightBytes]));
}

export function equalText(left, right) {
  const a = encodeText(left);
  const b = encodeText(right);
  if (a.length !== b.length) return false;
  for (let index = 0; index < a.length; index += 1) if (a[index] !== b[index]) return false;
  return true;
}

function hasLoneSurrogate(value) {
  for (let index = 0; index < value.length; index += 1) {
    const unit = value.charCodeAt(index);
    if (unit >= 0xd800 && unit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) return true;
      index += 1;
    } else if (unit >= 0xdc00 && unit <= 0xdfff) return true;
  }
  return false;
}

function asBytes(value) {
  if (!(value instanceof Uint8Array)) throw new Error("string_abi_v2.bytes_required");
  return value;
}

function checkedAdd(left, right) {
  const value = left + right;
  if (!Number.isSafeInteger(value)) throw new Error("string_abi_v2.size_overflow");
  return value;
}

function checkedMultiply(left, right) {
  const value = left * right;
  if (!Number.isSafeInteger(value)) throw new Error("string_abi_v2.size_overflow");
  return value;
}

function requestHeaderSize(types) {
  let size = 0;
  for (const type of types) {
    if (type === "bool") size = checkedAdd(size, 1);
    else if (type === "i64" || type === "string") size = checkedAdd(size, 8);
    else throw new Error("string_abi_v2.unsupported_type");
  }
  return size;
}
