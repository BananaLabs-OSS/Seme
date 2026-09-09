local SemeValues = dofile("reference/lua/seme-values.lua")
Seme = SemeValues
local program = arg[1]
local vectors_path = arg[2]
if program == nil or vectors_path == nil then error("usage: nvim -l lua-aggregate-native.lua PROGRAM.lua VECTORS.json") end
dofile(program)

local function vector(bias, second, slice_values, entries, key)
  local function i64(value) return Seme.i64(value) end
  local wrapped_entries = {}
  for index, entry in ipairs(entries) do wrapped_entries[index] = { key = i64(entry[1]), value = i64(entry[2]) } end
  local result = Observe(
    Seme.record("Bounds", { "bias" }, { bias = i64(bias) }),
    Seme.array(2, { i64("2"), i64(second) }),
    Seme.slice(vim.tbl_map(i64, slice_values)),
    Seme.map("i64", wrapped_entries),
    i64(key))
  return Seme.i64_decimal(result)
end

local handle = assert(io.open(vectors_path, "rb")); local vectors = vim.json.decode(handle:read("*a")); handle:close()
local observations = {}
for _, item in ipairs(vectors.valid) do
  observations[item.name] = vector(item.record, item.array[2], item.slice, item.map, item.key)
  assert(observations[item.name] == item.result)
end
local malformed = 0
local function rejects(build)
  if not pcall(build) then malformed = malformed + 1 end
end
rejects(function() return Observe(Seme.record("Bounds", { "bias" }, {}), Seme.array(2, { Seme.i64("0"), Seme.i64("0") }), Seme.slice({}), Seme.map("i64", {}), Seme.i64("0")) end)
rejects(function() return Observe(Seme.record("Bounds", { "bias" }, { bias = Seme.i64("0") }), Seme.array(2, { Seme.i64("0") }), Seme.slice({}), Seme.map("i64", {}), Seme.i64("0")) end)
rejects(function() return Observe(Seme.record("Bounds", { "bias" }, { bias = Seme.i64("0") }), Seme.array(2, { Seme.i64("0"), Seme.i64("0") }), Seme.slice({}), {}, Seme.i64("0")) end)
assert(malformed == 3)
io.write(vim.json.encode({ valid = observations, malformed = malformed }), "\n")
