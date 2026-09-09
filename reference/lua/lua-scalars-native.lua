local SemeValues = dofile("reference/lua/seme-values.lua"); Seme = SemeValues
if arg[1] == nil or arg[2] == nil then error("usage: nvim -l lua-scalars-native.lua PROGRAM.lua ENTRY") end
dofile(arg[1])
local observations = {}
if arg[2] == "ObserveI64" then
  observations.zero = Seme.i64_decimal(ObserveI64(Seme.i64("0")))
  observations.wrap = Seme.i64_decimal(ObserveI64(Seme.i64("9223372036854775807")))
  assert(observations.zero == "1" and observations.wrap == "-9223372036854775808")
elseif arg[2] == "ObserveBoolean" then
  observations.false_value, observations.true_value = ObserveBoolean(Seme.boolean(false)), ObserveBoolean(Seme.boolean(true))
  assert(not observations.false_value and observations.true_value)
elseif arg[2] == "ObserveText" then
  observations.empty = Seme.text_string(ObserveText(Seme.text("")))
  observations.unicode = Seme.text_string(ObserveText(Seme.text("世界🚀")))
  assert(observations.empty == "!" and observations.unicode == "世界🚀!")
else error("unknown entry") end
io.write(vim.json.encode(observations), "\n")
