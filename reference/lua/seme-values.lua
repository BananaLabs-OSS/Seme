-- Explicit immutable LuaJIT representations for canonical compound values.
-- Raw tables and nil are deliberately not accepted as implicit Seme values.
local Seme = {}
local storage = setmetatable({}, { __mode = "k" })

local function freeze(kind, value)
  local proxy = newproxy(true)
  storage[proxy] = { kind = kind, value = value }
  local meta = getmetatable(proxy)
  meta.__metatable = "seme.value"
  meta.__newindex = function() error("seme.value_immutable", 2) end
  meta.__tostring = function() return "seme." .. kind end
  return proxy
end

local function unpack_value(value, kind)
  local item = storage[value]
  if item == nil or item.kind ~= kind then error("seme.expected_" .. kind, 3) end
  return item.value
end

function Seme.kind(value)
  local item = storage[value]
  return item and item.kind or nil
end

function Seme.i64(decimal)
  if type(decimal) ~= "string" or not string.match(decimal, "^-?%d+$") then error("seme.invalid_i64", 2) end
  if decimal == "-0" or (#decimal > 1 and string.sub(decimal, 1, 1) == "0") or string.match(decimal, "^-0%d") then error("seme.invalid_i64", 2) end
  local negative = string.sub(decimal, 1, 1) == "-"
  local digits = negative and string.sub(decimal, 2) or decimal
  local limit = negative and "9223372036854775808" or "9223372036854775807"
  if #digits > #limit or (#digits == #limit and digits > limit) then error("seme.invalid_i64", 2) end
  return freeze("i64", decimal)
end

function Seme.text(value)
  if type(value) ~= "string" then error("seme.invalid_text", 2) end
  -- Native UTF-8 validation belongs to the provider boundary; this wrapper
  -- prevents text/bytes conflation and retains the exact byte sequence.
  return freeze("text", value)
end

function Seme.bytes(value)
  if type(value) ~= "string" then error("seme.invalid_bytes", 2) end
  return freeze("bytes", value)
end

function Seme.none()
  return freeze("option", { some = false })
end

function Seme.some(value)
  if storage[value] == nil then error("seme.untyped_some", 2) end
  return freeze("option", { some = true, value = value })
end

function Seme.option_is_some(option)
  return unpack_value(option, "option").some
end

function Seme.option_value(option)
  local item = unpack_value(option, "option")
  if not item.some then error("seme.option_none", 2) end
  return item.value
end

function Seme.ok(value)
  if storage[value] == nil then error("seme.untyped_ok", 2) end
  return freeze("result", { ok = true, value = value })
end

function Seme.err(value)
  if storage[value] == nil then error("seme.untyped_error", 2) end
  return freeze("result", { ok = false, value = value })
end

function Seme.result_is_ok(result) return unpack_value(result, "result").ok end
function Seme.result_value(result) return unpack_value(result, "result").value end

local function copy_values(values)
  if type(values) ~= "table" or getmetatable(values) ~= nil then error("seme.expected_plain_table", 3) end
  local copy = {}
  for index = 1, #values do
    if storage[values[index]] == nil then error("seme.untyped_element", 3) end
    copy[index] = values[index]
  end
  for key in pairs(values) do
    if type(key) ~= "number" or key < 1 or key > #values or key % 1 ~= 0 then error("seme.collection_hole_or_key", 3) end
  end
  return copy
end

function Seme.array(length, values, ...)
  if type(length) ~= "number" then
    local supplied = { length, values, ... }
    return freeze("array", copy_values(supplied))
  end
  if length < 1 or length % 1 ~= 0 then error("seme.invalid_array_length", 2) end
  local copy = copy_values(values)
  if #copy ~= length then error("seme.array_length", 2) end
  return freeze("array", copy)
end

function Seme.slice(values) return freeze("slice", copy_values(values)) end
function Seme.collection_length(value)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") then error("seme.expected_collection", 2) end
  return #item.value
end
function Seme.collection_get(value, zero_index)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") then error("seme.expected_collection", 2) end
  if type(zero_index) ~= "number" or zero_index % 1 ~= 0 or zero_index < 0 or zero_index >= #item.value then error("seme.index_out_of_range", 2) end
  return item.value[zero_index + 1]
end
Seme.length = Seme.collection_length
Seme.index_zero = Seme.collection_get

function Seme.record(type_name, field_names, values)
  if type(type_name) ~= "string" or type(field_names) ~= "table" or type(values) ~= "table" then error("seme.invalid_record", 2) end
  local fields, seen = {}, {}
  for index, name in ipairs(field_names) do
    if type(name) ~= "string" or seen[name] or storage[values[name]] == nil then error("seme.invalid_record_field", 2) end
    seen[name], fields[index] = true, { name = name, value = values[name] }
  end
  for name in pairs(values) do if not seen[name] then error("seme.extra_record_field", 2) end end
  return freeze("record", { type_name = type_name, fields = fields })
end
function Seme.record_get(record, name)
  for _, field in ipairs(unpack_value(record, "record").fields) do if field.name == name then return field.value end end
  error("seme.missing_record_field", 2)
end

local function key_token(key)
  local item = storage[key]
  if item == nil or (item.kind ~= "i64" and item.kind ~= "text" and item.kind ~= "bytes") then error("seme.invalid_map_key", 3) end
  return item.kind .. ":" .. #item.value .. ":" .. item.value
end
function Seme.map(value_kind, entries)
  if type(value_kind) ~= "string" then entries, value_kind = value_kind, nil end
  if type(entries) ~= "table" then error("seme.invalid_map", 2) end
  local copy, seen = {}, {}
  for _, entry in ipairs(entries) do
    if type(entry) ~= "table" or storage[entry.value] == nil then error("seme.invalid_map_entry", 2) end
    local token = key_token(entry.key)
    if seen[token] then error("seme.duplicate_map_key", 2) end
    seen[token], copy[#copy + 1] = true, { token = token, key = entry.key, value = entry.value }
  end
  table.sort(copy, function(left, right) return left.token < right.token end)
  return freeze("map", { value_kind = value_kind, entries = copy })
end
function Seme.map_lookup(map, key)
  local token = key_token(key)
  for _, entry in ipairs(unpack_value(map, "map").entries) do if entry.token == token then return Seme.some(entry.value) end end
  return Seme.none()
end
function Seme.empty_map(value_kind)
  if value_kind == nil then return freeze("map", { value_kind = nil, entries = {} }) end
  return Seme.map(value_kind, {})
end
function Seme.map_update(map, key, value)
  if storage[value] == nil then error("seme.untyped_map_value", 2) end
  local old = unpack_value(map, "map")
  local token, entries, replaced = key_token(key), {}, false
  for _, entry in ipairs(old.entries) do
    if entry.token == token then entries[#entries + 1], replaced = { key = key, value = value }, true
    else entries[#entries + 1] = { key = entry.key, value = entry.value } end
  end
  if not replaced then entries[#entries + 1] = { key = key, value = value } end
  return Seme.map(old.value_kind, entries)
end
function Seme.lookup_zero(map, key)
  local item, token = unpack_value(map, "map"), key_token(key)
  for _, entry in ipairs(item.entries) do if entry.token == token then return entry.value end end
  if item.value_kind == "i64" then return Seme.i64("0") end
  if item.value_kind == "text" then return Seme.text("") end
  if item.value_kind == "bytes" then return Seme.bytes("") end
  error("seme.map_zero_unsupported", 2)
end

return Seme
