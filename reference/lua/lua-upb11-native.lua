local root, adapter = assert(arg[1]), assert(arg[2])
package.path = root .. "/?.lua;"
local Seme = dofile(adapter); _G.Seme = Seme
local application, configuration, policy = require("application"), require("configuration"), require("policy")
local names = {}; for _, name in ipairs({ "InitializePolicy", "BuildPolicy" }) do if type(policy[name]) == "function" then names[#names + 1] = name end end
assert(#names == 1, "lua_upb11_native.identity_binding")
local input = Seme.record("ConfigInput", { "Enabled", "Limit", "Namespace" }, { Enabled = configuration.DefaultEnabled(), Limit = configuration.DefaultLimit(), Namespace = configuration.DefaultNamespace() })
local checked = configuration.ValidateLimit(Seme.field(input, "Limit")); assert(Seme.result_is_ok(checked), "validator")
local settings = configuration.InitializeConfig(input); assert(Seme.result_is_ok(settings), "configuration")
local selected = policy[names[1]](Seme.result_value(settings)); assert(Seme.result_is_ok(selected), "policy")
local runtime = application.Assemble(Seme.result_value(settings), Seme.result_value(selected)); assert(Seme.result_is_ok(runtime), "application")
local value = Seme.result_value(runtime); assert(Seme.boolean(Seme.field(value, "Ready")) and Seme.i64_decimal(Seme.field(value, "Total")) == "16" and Seme.text_string(Seme.field(value, "Namespace")) == "seme", "runtime")
io.write(vim.json.encode({ lifecycle = { "configuration", "policy", "application" }, ready = true, total = "16", namespace = "seme" }), "\n")
