local root, vectors, adapter = assert(arg[1]), assert(arg[2]), assert(arg[3])
package.path = root .. "/?.lua;"
local Seme = dofile(adapter)
_G.Seme = Seme
local controlled = require("controlled")
local file = assert(io.open(vectors, "r")); local data = vim.json.decode(file:read("*a")); file:close()
for _, item in ipairs(data.valid) do
  local arguments = item.arguments
  local state = Seme.record("ControlledState", { "Value" }, { Value = Seme.i64_literal(arguments[1].fields.Value.i64) })
  local command = Seme.record("ControlledCommand", { "Delta" }, { Delta = Seme.i64_literal(arguments[2].fields.Delta.i64) })
  local trace = {}; _G.Seme_observability_log = function(value) trace[#trace + 1] = value end
  local result = controlled.DispatchControlled(state, command); _G.Seme_observability_log = nil
  assert(#trace == 1 and trace[1] == true, "effect trace")
  io.write(vim.json.encode({ value = { kind = "record", fields = { Value = { kind = "i64", i64 = Seme.i64_decimal(Seme.field(result, "Value")) } } }, effects = {{ capability = "observability.log", value = true }} }), "\n")
end
