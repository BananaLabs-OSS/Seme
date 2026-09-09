local Seme = dofile("reference/lua/seme-values.lua")

assert(Seme.boolean(false) == false)
assert(Seme.boolean(true) == true)
for _, value in ipairs({ 0, 1, "", "false", {}, function() end }) do
  assert(not pcall(Seme.boolean, value), "Lua truthiness crossed the canonical Boolean boundary")
end
