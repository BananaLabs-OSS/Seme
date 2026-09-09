local SemeValues = dofile("reference/lua/seme-values.lua")
Seme = SemeValues
if arg[1] == nil or arg[2] == nil then error("usage: nvim -l lua-uab-03-native.lua PROGRAM.lua VECTORS.json") end
dofile(arg[1])
local handle = assert(io.open(arg[2], "rb"))
local vectors = vim.json.decode(handle:read("*a"))
handle:close()
local observed = {}
for _, item in ipairs(vectors.valid) do
  local converted = {}; for index, value in ipairs(item.values) do converted[index] = Seme.i64(value) end
  local values = Seme.slice(converted)
  local result = Accumulate(Seme.i64(item.limit), item.enabled, values, Seme.i64(item.probe), item.bypass)
  observed[item.name] = Seme.i64_decimal(result)
  assert(observed[item.name] == item.result)
end
local malformed=0
for _, item in ipairs(vectors.malformed) do
  local call
  if item.category == "checked-index" then call = function() Accumulate(Seme.i64(item.limit), item.enabled, Seme.slice({}), Seme.i64(item.probe), item.bypass) end
  elseif item.category == "limit-text" then call = function() Accumulate("4", true, Seme.slice({Seme.i64("99")}), Seme.i64("0"), false) end
  elseif item.category == "condition-i64" then call = function() Accumulate(Seme.i64("4"), Seme.i64("1"), Seme.slice({Seme.i64("99")}), Seme.i64("0"), false) end
  elseif item.category == "limit-overflow" then call = function() Accumulate(Seme.i64("9223372036854775808"), true) end
  else error("lua.uab03_unknown_malformed_category:" .. tostring(item.category)) end
  if not pcall(call) then malformed=malformed+1 end
end
assert(malformed==#vectors.malformed);io.write(vim.json.encode({valid=observed,malformed=malformed}), "\n")
