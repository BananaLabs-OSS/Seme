local program, vectors, family = arg[1], arg[2], arg[3]
if not program or not vectors or not family then error("usage: lua-uab-06-native PROGRAM VECTORS FAMILY") end
local directory = assert(debug.getinfo(1, "S").source:match("^@(.*/)") )
_G.Seme = dofile(directory .. "seme-values.lua")
dofile(program)
local corpus = vim.json.decode(table.concat(vim.fn.readfile(vectors), "\n"))
local fn = family == "immutable" and Immutable or Mutable
local valid = {}
for _, item in ipairs(corpus[family]) do
  local arguments = {}
  for index, value in ipairs(item.arguments) do arguments[index] = Seme.i64(value) end
  local observed = Seme.i64_decimal(fn(unpack(arguments)))
  if observed ~= item.result then error("lua.uab06.native:" .. item.name) end
  valid[item.name] = observed
end
print(vim.json.encode({ valid = valid }))
