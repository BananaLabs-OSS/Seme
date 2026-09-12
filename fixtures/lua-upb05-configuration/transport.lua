local Seme = assert(_G.Seme, "Seme adapters are required")

---@class TransportCommand
---@field Sequence seme.i64
---@field Value seme.i64

---@class TransportEvent
---@field Sequence seme.i64
---@field Value seme.i64

---@seme-id 8055555555555555555555555555550c
---@param command TransportCommand
---@return TransportEvent
local function Dispatch(command)
  return Seme.record("TransportEvent", { "Sequence", "Value" }, { Sequence = Seme.field(command, "Sequence"), Value = Seme.add(Seme.field(command, "Value"), Seme.i64_literal("1")) })
end

---@seme-id 8055555555555555555555555555550d
---@param event TransportEvent
---@return seme.i64
local function Replay(event)
  return Seme.field(event, "Value")
end

return { Dispatch = Dispatch, Replay = Replay }
