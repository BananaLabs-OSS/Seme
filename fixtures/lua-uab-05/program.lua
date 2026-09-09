local Seme = assert(_G.Seme, "Seme adapters are required")

local Adjuster = Seme.protocol("Adjuster", { "adjust" })

---@class Adjustment
---@field amount seme.i64

---@param receiver Adjustment
---@param value seme.i64
---@return seme.i64
local function Offset_adjust(receiver, value)
  return Seme.add(value, Seme.field(receiver, "amount"))
end

---@param receiver Adjustment
---@param value seme.i64
---@return seme.i64
local function DoubleOffset_adjust(receiver, value)
  return Seme.add(Seme.add(value, Seme.field(receiver, "amount")), Seme.field(receiver, "amount"))
end

local OffsetAdjuster = Seme.implementation(Adjuster, "record", {
  adjust = Offset_adjust,
})
local DoubleOffsetAdjuster = Seme.implementation(Adjuster, "record", {
  adjust = DoubleOffset_adjust,
})

---@param useDouble boolean
---@param amount seme.i64
---@param value seme.i64
---@return seme.i64
function Dispatch(useDouble, amount, value)
  return Seme.protocol_dispatch(useDouble, OffsetAdjuster, DoubleOffsetAdjuster, "adjust", "Adjustment", "amount", amount, value)
end
