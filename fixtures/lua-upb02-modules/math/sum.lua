local Seme = assert(_G.Seme, "Seme adapters are required")

---@seme-id 80222222222222222222222222222201
---@param value seme.i64
---@return seme.i64
local function normalize(value)
  return Seme.add(value, Seme.i64_literal("0"))
end

---@seme-id 80222222222222222222222222222202
---@param left seme.i64
---@param right seme.i64
---@return seme.i64
local function Sum(left, right)
  return Seme.add(left, right)
end

return { Sum = Sum }
