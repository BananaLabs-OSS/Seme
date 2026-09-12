local Seme = assert(_G.Seme, "Seme adapters are required")

---@class Runtime
---@field Ready boolean
---@field Total seme.i64
---@field Namespace seme.text

---@seme-id 80555555555555555555555555555507
---@param settings Settings
---@param policy Policy
---@return seme.result<Runtime,seme.text>
local function Assemble(settings, policy)
  return Seme.ok(Seme.record("Runtime", { "Ready", "Total", "Namespace" }, { Ready = Seme.field(settings, "Enabled"), Total = Seme.add(Seme.field(settings, "Limit"), Seme.field(policy, "Limit")), Namespace = Seme.field(settings, "Namespace") }))
end

---@seme-id 80555555555555555555555555555508
---@param enabled boolean
---@param limit seme.i64
---@return seme.i64
local function Run(enabled, limit)
  if Seme.boolean(enabled) then
    return Seme.add(limit, limit)
  else
    return limit
  end
end

return { Assemble = Assemble, Run = Run }
