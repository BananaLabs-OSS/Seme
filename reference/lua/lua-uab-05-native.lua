local program, vectors = arg[1], arg[2]
if not program or not vectors then error("usage: lua-uab-05-native PROGRAM VECTORS") end
local directory = assert(debug.getinfo(1, "S").source:match("^@(.*/)"))
_G.Seme = dofile(directory .. "seme-values.lua")
dofile(program)
local corpus = vim.json.decode(table.concat(vim.fn.readfile(vectors), "\n"))
local valid = {}
for _, item in ipairs(corpus.valid) do
  local result = Dispatch(item.use_double, Seme.i64(item.amount), Seme.i64(item.value))
  local observed = Seme.i64_decimal(result)
  if observed ~= item.result then error("lua.uab05.native:" .. item.name) end
  valid[item.name] = observed
end
print(vim.json.encode({ valid = valid }))
