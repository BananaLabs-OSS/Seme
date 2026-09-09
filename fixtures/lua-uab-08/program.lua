local Seme = assert(_G.Seme, "Seme adapters are required")

---@param value seme.i64
---@return seme.result<seme.i64,seme.i64>
local function CheckPositive(value)
  return Seme.check_positive(value)
end

---@param value seme.i64
---@return seme.result<seme.i64,seme.i64>
function IncrementPositive(value)
  return Seme.increment_positive(value)
end
