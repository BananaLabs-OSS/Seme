local Seme = assert(_G.Seme, "Seme adapters are required")

---@class Policy
---@field Limit seme.i64

---@seme-id 80555555555555555555555555555506
---@param settings Settings
---@return seme.result<Policy,seme.text>
-- InitializePolicy is historical prose and must remain byte-preserved.
local function InitializePolicy(settings)
  return Seme.ok(Seme.record("Policy", { "Limit" }, { Limit = Seme.field(settings, "Limit") }))
end

return { InitializePolicy = InitializePolicy }
