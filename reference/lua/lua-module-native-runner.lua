local root, vectors = assert(arg[1]), assert(arg[2])
package.path = root .. "/?.lua;" .. root .. "/?/init.lua"
local runtime = dofile(assert(arg[3]))
_G.Seme = runtime
local app = require("application")
local input = assert(io.open(vectors, "r")); local data = vim.json.decode(input:read("*a")); input:close()
local valid = {}
for _, item in ipairs(data.valid) do
  local observed = runtime.i64_decimal(app.Run(runtime.i64_literal(item.arguments[1].i64), runtime.i64_literal(item.arguments[2].i64)))
  assert(observed == item.result.i64, item.name); valid[item.name] = observed
end
io.write(vim.json.encode({ valid = valid }), "\n")
