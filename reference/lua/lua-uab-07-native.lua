local program,vectors=arg[1],arg[2];if not program or not vectors then error("usage: lua-uab-07-native PROGRAM VECTORS") end
local directory=assert(debug.getinfo(1,"S").source:match("^@(.*/)") );_G.Seme=dofile(directory.."seme-values.lua");dofile(program)
local corpus,valid=vim.json.decode(table.concat(vim.fn.readfile(vectors),"\n")),{}
for _,item in ipairs(corpus.valid) do
  local counter = Seme.record("Counter", {"Value"}, {Value=Seme.i64(item.value)})
  local observed = Step(counter, Seme.i64(item.delta))
  local state = Seme.record_get(Seme.transition_state(observed), "Value")
  local result = Seme.transition_result(observed)
  local actual = {state=Seme.i64_decimal(state), result=Seme.i64_decimal(result)}
  if actual.state~=item.state or actual.result~=item.result then error("lua.uab07:"..item.name) end
  valid[item.name]=actual
end
print(vim.json.encode({valid=valid}))
