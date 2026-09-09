-- Explicit immutable LuaJIT representations for canonical compound values.
-- Raw tables and nil are deliberately not accepted as implicit Seme values.
local Seme = {}
local storage = setmetatable({}, { __mode = "k" })

-- Explicit adapted boundary for canonical Boolean. Lua truthiness is broader:
-- values such as 0 and "" are truthy, so they must never cross implicitly.
function Seme.boolean(value)
  if type(value) ~= "boolean" then error("seme.expected_boolean", 2) end
  return value
end

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
local function operand(value)
  if type(value) == "number" and value >= 0 and value % 1 == 0 then return tostring(value) end
  return unpack_value(value, "i64")
end
local function native_i64(decimal)
  local ffi = require("ffi")
  local negative, start = string.sub(decimal, 1, 1) == "-", 1
  if negative then start = 2 end
  local result = ffi.new("int64_t", 0)
  for index = start, #decimal do
    local digit = string.byte(decimal, index) - string.byte("0")
    result = negative and (result * 10 - digit) or (result * 10 + digit)
  end
  return result
end
function Seme.add(left, right)
  local left_value, right_value = operand(left), operand(right)
  local result = native_i64(left_value) + native_i64(right_value)
  return Seme.i64((string.gsub(tostring(result), "LL$", "")))
end
function Seme.multiply(left, right)
  local left_value, right_value = operand(left), operand(right)
  local result = native_i64(left_value) * native_i64(right_value)
  return Seme.i64((string.gsub(tostring(result), "LL$", "")))
end
function Seme.subtract(left, right)
  local left_value, right_value = operand(left), operand(right)
  local result = native_i64(left_value) - native_i64(right_value)
  return Seme.i64((string.gsub(tostring(result), "LL$", "")))
end
function Seme.less_equal(left, right) return native_i64(operand(left)) <= native_i64(operand(right)) end
function Seme.equal_i64(left, right) return native_i64(operand(left)) == native_i64(operand(right)) end
function Seme.i64_decimal(value) return unpack_value(value, "i64") end
Seme.i64_literal = Seme.i64

function Seme.text(value)
  if type(value) ~= "string" then error("seme.invalid_text", 2) end
  -- Native UTF-8 validation belongs to the provider boundary; this wrapper
  -- prevents text/bytes conflation and retains the exact byte sequence.
  return freeze("text", value)
end
function Seme.text_string(value) return unpack_value(value, "text") end
function Seme.text_concat(left, right)
  local left_value = storage[left] and unpack_value(left, "text") or left
  local right_value = storage[right] and unpack_value(right, "text") or right
  if type(left_value) ~= "string" or type(right_value) ~= "string" then error("seme.expected_text", 2) end
  return Seme.text(left_value .. right_value)
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

function Seme.slice(values, ...)
  if type(values) == "table" then return freeze("slice", copy_values(values)) end
  if values == nil then return freeze("slice", {}) end
  return freeze("slice", copy_values({ values, ... }))
end
function Seme.collection_length(value)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") then error("seme.expected_collection", 2) end
  return #item.value
end
function Seme.collection_get(value, zero_index)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") then error("seme.expected_collection", 2) end
  if storage[zero_index] and storage[zero_index].kind == "i64" then zero_index = tonumber(unpack_value(zero_index, "i64")) end
  if type(zero_index) ~= "number" or zero_index % 1 ~= 0 or zero_index < 0 or zero_index >= #item.value then error("seme.index_out_of_range", 2) end
  return item.value[zero_index + 1]
end
Seme.length = Seme.collection_length
function Seme.length_i64(value) return Seme.i64(tostring(Seme.collection_length(value))) end
Seme.index_zero = Seme.collection_get
function Seme.collection_append(value, element)
  local item = storage[value]
  if item == nil or item.kind ~= "slice" or storage[element] == nil then error("seme.expected_slice_append", 2) end
  local copy = {}; for index, old in ipairs(item.value) do copy[index] = old end
  copy[#copy + 1] = element
  return freeze("slice", copy)
end
function Seme.collection_update(value, zero_index, element)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") or storage[element] == nil then error("seme.expected_collection_update", 2) end
  if storage[zero_index] and storage[zero_index].kind == "i64" then zero_index = tonumber(unpack_value(zero_index, "i64")) end
  if type(zero_index) ~= "number" or zero_index % 1 ~= 0 or zero_index < 0 or zero_index >= #item.value then error("seme.index_out_of_range", 2) end
  local copy = {}; for index, old in ipairs(item.value) do copy[index] = old end
  copy[zero_index + 1] = element
  return freeze(item.kind, copy)
end
function Seme.slice_remove(value, zero_index)
  local item = storage[value]
  if item == nil or item.kind ~= "slice" then error("seme.expected_slice_remove", 2) end
  if storage[zero_index] and storage[zero_index].kind == "i64" then zero_index = tonumber(unpack_value(zero_index, "i64")) end
  if type(zero_index) ~= "number" or zero_index % 1 ~= 0 or zero_index < 0 or zero_index >= #item.value then error("seme.index_out_of_range", 2) end
  local copy = {}; for index, old in ipairs(item.value) do if index ~= zero_index + 1 then copy[#copy + 1] = old end end
  return freeze("slice", copy)
end
function Seme.fold(value, initial, callback)
  local item = storage[value]
  if item == nil or (item.kind ~= "array" and item.kind ~= "slice") or storage[initial] == nil or type(callback) ~= "function" then error("seme.invalid_fold", 2) end
  local accumulator = initial
  for index = 1, #item.value do
    accumulator = callback(accumulator, item.value[index])
    if storage[accumulator] == nil then error("seme.untyped_fold_result", 2) end
  end
  return accumulator
end

function Seme.record(type_name, field_names, values)
  if type(type_name) ~= "string" or type(field_names) ~= "table" or type(values) ~= "table" then error("seme.invalid_record", 2) end
  local fields, seen = {}, {}
  for index, name in ipairs(field_names) do
    if type(name) ~= "string" or seen[name] or (storage[values[name]] == nil and type(values[name]) ~= "boolean") then error("seme.invalid_record_field", 2) end
    seen[name], fields[index] = true, { name = name, value = values[name] }
  end
  for name in pairs(values) do if not seen[name] then error("seme.extra_record_field", 2) end end
  return freeze("record", { type_name = type_name, fields = fields })
end
function Seme.record_get(record, name)
  for _, field in ipairs(unpack_value(record, "record").fields) do if field.name == name then return field.value end end
  error("seme.missing_record_field", 2)
end
Seme.field = Seme.record_get

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
function Seme.map_remove(map, key)
  local old, token = unpack_value(map, "map"), key_token(key)
  local entries = {}
  for _, entry in ipairs(old.entries) do
    if entry.token ~= token then entries[#entries + 1] = { key = entry.key, value = entry.value } end
  end
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
function Seme.bytes_literal(value) return Seme.bytes(value) end
function Seme.bytes_equal(left, right)
  return unpack_value(left, "bytes") == unpack_value(right, "bytes")
end
function Seme.text_equal(left, right)
  local left_value = storage[left] and unpack_value(left, "text") or left
  local right_value = storage[right] and unpack_value(right, "text") or right
  if type(left_value) ~= "string" or type(right_value) ~= "string" then error("seme.expected_text", 2) end
  return left_value == right_value
end
function Seme.match_option(option, none_value, some)
  local item = unpack_value(option, "option")
  if not item.some then return none_value end
  if type(some) ~= "function" then error("seme.expected_match_callback", 2) end
  return some(item.value)
end
function Seme.match_result(result, ok, error_)
  local item = unpack_value(result, "result")
  local callback = item.ok and ok or error_
  if type(callback) ~= "function" then error("seme.expected_match_callback", 2) end
  return callback(item.value)
end

-- Protocol values are explicit Seme adapters. They deliberately do not infer
-- contracts from ordinary Lua tables, metatables, `__index`, or method names.
function Seme.protocol(name, requirements)
  if type(name) ~= "string" or name == "" or type(requirements) ~= "table" then error("seme.invalid_protocol", 2) end
  local copy, seen = {}, {}
  for index, requirement in ipairs(requirements) do
    if index > 32 or type(requirement) ~= "string" or requirement == "" or seen[requirement] then error("seme.invalid_protocol_requirement", 2) end
    seen[requirement], copy[index] = true, requirement
  end
  if #copy == 0 then error("seme.empty_protocol", 2) end
  return freeze("protocol", { name = name, requirements = copy })
end
function Seme.implementation(protocol, concrete_kind, methods)
  local contract = unpack_value(protocol, "protocol")
  if type(concrete_kind) ~= "string" or concrete_kind == "" or type(methods) ~= "table" then error("seme.invalid_implementation", 2) end
  local copy, count = {}, 0
  for _, requirement in ipairs(contract.requirements) do
    if type(methods[requirement]) ~= "function" then error("seme.missing_protocol_method", 2) end
    copy[requirement], count = methods[requirement], count + 1
  end
  for name in pairs(methods) do
    if copy[name] == nil then error("seme.extra_protocol_method", 2) end
  end
  if count > 32 then error("seme.implementation_too_large", 2) end
  return freeze("implementation", { protocol = protocol, concrete_kind = concrete_kind, methods = copy })
end
function Seme.interface_value(implementation, value)
  local witness = unpack_value(implementation, "implementation")
  if Seme.kind(value) ~= witness.concrete_kind then error("seme.interface_concrete_kind", 2) end
  return freeze("interface", { implementation = implementation, value = value })
end
function Seme.dynamic_call(interface, requirement, ...)
  local boxed = unpack_value(interface, "interface")
  local witness = unpack_value(boxed.implementation, "implementation")
  local contract = unpack_value(witness.protocol, "protocol")
  if type(requirement) ~= "string" or witness.methods[requirement] == nil then error("seme.unknown_protocol_method", 2) end
  local declared = false
  for _, name in ipairs(contract.requirements) do if name == requirement then declared = true end end
  if not declared then error("seme.unknown_protocol_method", 2) end
  return witness.methods[requirement](boxed.value, ...)
end
Seme.box_protocol = Seme.interface_value
Seme.protocol_call = Seme.dynamic_call
function Seme.protocol_dispatch(condition, when_false, when_true, requirement, record_name, field_name, receiver_value, argument)
  if type(condition) ~= "boolean" then error("seme.expected_boolean", 2) end
  if type(record_name) ~= "string" or type(field_name) ~= "string" then error("seme.invalid_dispatch_record", 2) end
  local implementation = condition and when_true or when_false
  local receiver = Seme.record(record_name, { field_name }, { [field_name] = receiver_value })
  return Seme.dynamic_call(Seme.interface_value(implementation, receiver), requirement, argument)
end
function Seme.protocol_dispatch_value(condition, when_false, when_true, requirement, receiver_value, argument)
  if type(condition) ~= "boolean" or type(requirement) ~= "string" then error("seme.invalid_protocol_dispatch", 2) end
  local implementation = condition and when_true or when_false
  return Seme.dynamic_call(Seme.interface_value(implementation, receiver_value), requirement, argument)
end

function Seme.immutable_closure_run(base, value)
  local captured = base
  local closure = function(argument) return Seme.add(captured, argument) end
  return closure(value)
end
function Seme.closure(initial, body)
  if type(body) ~= "function" then error("seme.expected_closure_body", 2) end
  return function(value) return body(initial, value) end
end
function Seme.mutable_closure(initial, body)
  if type(body) ~= "function" then error("seme.expected_closure_body", 2) end
  local captured = initial
  return function(value) captured = body(captured, value); return captured end
end

function Seme.mutable_closure_run(start, first, second)
  local captured = start
  local closure = function(delta) captured = Seme.add(captured, delta); return captured end
  closure(first)
  return closure(second)
end

function Seme.transition_step(counter, delta, field_name)
  if type(field_name) ~= "string" or not field_name:match("^[A-Za-z_][A-Za-z0-9_]*$") then error("seme.invalid_transition_field") end
  local value = Seme.record_get(counter, field_name)
  local transition = { state = Seme.record("Counter", { field_name }, { [field_name] = Seme.add(value, delta) }), result = value }
  return freeze("transition", transition)
end
function Seme.transition_state(value) return unpack_value(value, "transition").state end
function Seme.transition_result(value) return unpack_value(value, "transition").result end
function Seme.transition(state, result)
  if storage[state] == nil or storage[result] == nil then error("seme.untyped_transition", 2) end
  return freeze("transition", { state = state, result = result })
end
function Seme.check_positive(value)
  if Seme.less_equal(Seme.i64("0"), value) and Seme.i64_decimal(value) ~= "0" then return Seme.ok(value) end
  return Seme.err(Seme.i64("99"))
end
function Seme.increment_positive(value)
  return Seme.match_result(Seme.check_positive(value), function(accepted) return Seme.ok(Seme.add(accepted, Seme.i64("1"))) end, function(error_) return Seme.err(error_) end)
end
function Seme.observe(value)
  value = Seme.boolean(value)
  local capability = rawget(_G, "Seme_observability_log")
  if type(capability) ~= "function" then error("seme.capability_absent", 2) end
  capability(value)
end

return Seme
