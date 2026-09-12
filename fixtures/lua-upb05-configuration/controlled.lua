local Seme = assert(_G.Seme, "Seme adapters are required")

---@class ClockSample
---@field UnixMilliseconds seme.i64
---@field Sequence seme.i64
---@class RandomState
---@field Value seme.i64
---@class Draw
---@field State seme.i64
---@field Value seme.i64
---@class ControlledCommand
---@field Delta seme.i64
---@class ControlledState
---@field Value seme.i64
---@class ControlledResult
---@field Value seme.i64

---@seme-id 8055555555555555555555555555550e
---@param sample ClockSample
---@return seme.i64
local function ObserveClock(sample)
  return Seme.field(sample, "UnixMilliseconds")
end

---@seme-id 8055555555555555555555555555550f
---@param state RandomState
---@return Draw
local function Next(state)
  local value = Seme.add(Seme.multiply(Seme.field(state, "Value"), Seme.i64_literal("48271")), Seme.i64_literal("1"))
  return Seme.record("Draw", { "State", "Value" }, { State = value, Value = value })
end

---@seme-id 80555555555555555555555555555510
---@param state ControlledState
---@param command ControlledCommand
---@return ControlledResult
local function DispatchControlled(state, command)
  Seme.observe(true)
  return Seme.record("ControlledResult", { "Value" }, { Value = Seme.add(Seme.field(state, "Value"), Seme.field(command, "Delta")) })
end

---@seme-id 80555555555555555555555555555511
---@param state ControlledState
---@param command ControlledCommand
---@return ControlledResult
local function ReplayControlled(state, command)
  return Seme.record("ControlledResult", { "Value" }, { Value = Seme.add(Seme.field(state, "Value"), Seme.field(command, "Delta")) })
end

return { ObserveClock = ObserveClock, Next = Next, DispatchControlled = DispatchControlled, ReplayControlled = ReplayControlled }
