local Seme = assert(_G.Seme, "Seme adapters are required")
local math = require("math.sum")

---@seme-id 80333333333333333333333333333303
---@param base seme.i64
---@param delta seme.i64
---@return seme.i64
function Run(base, delta)
  return math.Sum(base, delta)
end

return { Run = Run }
