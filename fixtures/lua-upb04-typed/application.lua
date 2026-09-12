local Seme = assert(_G.Seme, "Seme adapters are required")
local collections = require("policy.collections")

---@seme-id 80444444444444444444444444444402
---@param first seme.i64
---@param second seme.i64
---@param index seme.i64
---@param replacement seme.i64
---@return seme.i64
local function Run(first, second, index, replacement)
  return collections.ReplaceAndRead(Seme.slice(first, second), index, replacement)
end

return { Run = Run }
