local Seme = assert(_G.Seme, "Seme adapters are required")

---@class V1
---@field Count seme.i64

---@class V2
---@field Count seme.i64
---@field Namespace seme.text

---@seme-id 80555555555555555555555555555509
---@param value V1
---@return seme.result<V1,seme.i64>
local function ValidateV1(value)
  return Seme.ok(value)
end

---@seme-id 8055555555555555555555555555550a
---@param value V2
---@return seme.result<V2,seme.i64>
local function ValidateV2(value)
  return Seme.ok(value)
end

---@seme-id 8055555555555555555555555555550b
---@param value V1
---@return seme.result<V2,seme.i64>
local function MigrateV1ToV2(value)
  return Seme.ok(Seme.record("V2", { "Count", "Namespace" }, { Count = Seme.field(value, "Count"), Namespace = "migrated" }))
end

return { ValidateV1 = ValidateV1, ValidateV2 = ValidateV2, MigrateV1ToV2 = MigrateV1ToV2 }
