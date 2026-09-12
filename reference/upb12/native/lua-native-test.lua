local root = assert(arg[1]); package.path = root .. "/?.lua"; _G.Seme = dofile(root .. "/seme-values.lua")
local controlled = require("controlled")
for i = 0, 4095 do
  local state, delta = Seme.i64_literal(tostring(i * 7919 - 10000000)), Seme.i64_literal(tostring(i * 104729 - 200000000))
  if i % 257 == 0 then state, delta = Seme.i64_literal("9223372036854775807"), Seme.i64_literal("1") end
  local trace = {}; _G.Seme_observability_log = function(value) trace[#trace + 1] = value end
  local result = controlled.DispatchControlled(Seme.record("ControlledState", { "Value" }, { Value = state }), Seme.record("ControlledCommand", { "Delta" }, { Delta = delta }))
  _G.Seme_observability_log = nil
  assert(Seme.i64_decimal(Seme.field(result, "Value")) == Seme.i64_decimal(Seme.add(state, delta)), "observation " .. i)
  assert(#trace == 1 and trace[1] == true, "effect " .. i)
end
io.write("Lua UPB12 native project passes 4,096 observations\n")
