local root, vectors, adapter = assert(arg[1]), assert(arg[2]), assert(arg[3])
package.path = root .. "/?.lua;"
local Seme = dofile(adapter)
_G.Seme = Seme
local transport = require("transport")
local file = assert(io.open(vectors, "r"))
local data = vim.json.decode(file:read("*a"))
file:close()
local valid, rejected = {}, {}
local function decode(item)
  local fields = item.arguments[1].fields
  if not fields.Sequence or fields.Sequence.kind ~= "i64" or not fields.Value or fields.Value.kind ~= "i64" then error("native.invalid_command") end
  return Seme.record("TransportCommand", { "Sequence", "Value" }, { Sequence = Seme.i64_literal(fields.Sequence.i64), Value = Seme.i64_literal(fields.Value.i64) })
end
local function encode(value)
  return { kind = "record", fields = { Sequence = { kind = "i64", i64 = Seme.i64_decimal(Seme.field(value, "Sequence")) }, Value = { kind = "i64", i64 = Seme.i64_decimal(Seme.field(value, "Value")) } } }
end
for _, item in ipairs(data.valid) do valid[item.name] = encode(transport.Dispatch(decode(item))) end
for _, item in ipairs(data.malformed) do local ok = pcall(function() transport.Dispatch(decode(item)) end); assert(not ok, item.name); rejected[item.name] = true end
io.write(vim.json.encode({ valid = valid, malformed = #data.malformed, rejected = rejected }), "\n")
