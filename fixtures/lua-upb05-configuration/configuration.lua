local Seme = assert(_G.Seme, "Seme adapters are required")

---@class ConfigInput
---@field Enabled boolean
---@field Limit seme.i64
---@field Namespace seme.text

---@class Settings
---@field Enabled boolean
---@field Limit seme.i64
---@field Namespace seme.text

---@seme-id 80555555555555555555555555555501
---@return boolean
local function DefaultEnabled()
  return Seme.bool_literal(true)
end

---@seme-id 80555555555555555555555555555502
---@return seme.i64
local function DefaultLimit()
  return Seme.i64_literal("8")
end

---@seme-id 80555555555555555555555555555503
---@return seme.text
local function DefaultNamespace()
  return "seme"
end

---@seme-id 80555555555555555555555555555504
---@param value seme.i64
---@return seme.result<seme.i64,seme.text>
local function ValidateLimit(value)
  return Seme.ok(value)
end

---@seme-id 80555555555555555555555555555505
---@param input ConfigInput
---@return seme.result<Settings,seme.text>
local function InitializeConfig(input)
  return Seme.ok(Seme.record("Settings", { "Enabled", "Limit", "Namespace" }, { Enabled = Seme.field(input, "Enabled"), Limit = Seme.field(input, "Limit"), Namespace = Seme.field(input, "Namespace") }))
end

return { DefaultEnabled = DefaultEnabled, DefaultLimit = DefaultLimit, DefaultNamespace = DefaultNamespace, ValidateLimit = ValidateLimit, InitializeConfig = InitializeConfig }
