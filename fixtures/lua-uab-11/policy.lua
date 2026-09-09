local Seme = assert(_G.Seme, "Seme adapters are required")

local Policy = Seme.protocol("Policy", { "adjust" })

---@seme-id 80111111111111111111111111111101
---@param receiver seme.i64
---@param value seme.i64
---@return seme.i64
local function Offset(receiver, value)
  return Seme.add(value, receiver)
end

---@seme-id 80111111111111111111111111111102
---@param receiver seme.i64
---@param value seme.i64
---@return seme.i64
local function Scale(receiver, value)
  return Seme.multiply(value, receiver)
end

local OffsetPolicy = Seme.implementation(Policy, "i64", {
  adjust = Offset,
})
local ScalePolicy = Seme.implementation(Policy, "i64", {
  adjust = Scale,
})

---@seme-id 80111111111111111111111111111103
---@param scale boolean
---@param delta seme.i64
---@param value seme.i64
---@return seme.i64
function ApplyPolicy(scale, delta, value)
  return Seme.protocol_dispatch_value(scale, OffsetPolicy, ScalePolicy, "adjust", delta, value)
end
