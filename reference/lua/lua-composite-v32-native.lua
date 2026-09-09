local Seme = dofile("reference/lua/seme-values.lua")
_G.Seme = Seme

if arg[1] == nil then error("usage: nvim -l lua-composite-v32-native.lua PROGRAM.lua") end
dofile(arg[1])

local valid = {
  none = Check(Seme.none()),
  ok = Check(Seme.some(Seme.ok(Seme.bytes("ok")))),
  wrong_bytes = Check(Seme.some(Seme.ok(Seme.bytes("no")))),
  error = Check(Seme.some(Seme.err(Seme.text("bad")))),
  wrong_error = Check(Seme.some(Seme.err(Seme.text("no"))))
}
assert(valid.none == false and valid.ok == true and valid.wrong_bytes == false)
assert(valid.error == true and valid.wrong_error == false)
local malformed = 0
for _, value in ipairs({ {}, { tag = "some" }, { tag = "unknown" } }) do
  if not pcall(Check, value) then malformed = malformed + 1 end
end
assert(malformed == 3)
io.write(vim.json.encode({ valid = valid, malformed = malformed }), "\n")
