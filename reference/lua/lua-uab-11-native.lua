local adapter, policy, application, corpus_path = assert(arg[1]), assert(arg[2]), assert(arg[3]), assert(arg[4])
local Seme = dofile(adapter)
_G.Seme = Seme
assert(loadfile(policy))()
assert(loadfile(application))()

local function i64(value) return Seme.i64(tostring(value)) end
local function state(name, values, counters)
  local elements = {}; for index, value in ipairs(values) do elements[index] = i64(value) end
  local entries = {}; for index, pair in ipairs(counters) do entries[index] = { key = i64(pair[1]), value = i64(pair[2]) } end
  return Seme.record("State", { "Name", "Values", "Counters" }, {
    Name = Seme.text(name), Values = Seme.slice(elements), Counters = Seme.map("i64", entries),
  })
end
local function command(key, index, delta, amount, scale)
  return Seme.record("Command", { "Key", "Index", "Delta", "Amount", "Scale" }, {
    Key = i64(key), Index = i64(index), Delta = i64(delta), Amount = i64(amount), Scale = Seme.boolean(scale),
  })
end
local function decimal(value) return Seme.i64_decimal(value) end
local function observe(initial, cmd, capability)
  local trace = {}
  if capability == false then _G.Seme_observability_log = nil
  else _G.Seme_observability_log = function(value) trace[#trace + 1] = value end end
  local ok, result = pcall(Apply, initial, cmd)
  return ok, result, trace
end
local function assert_error(initial, cmd, code)
  local ok, result, trace = observe(initial, cmd, true)
  assert(ok and not Seme.result_is_ok(result) and decimal(Seme.result_value(result)) == code and #trace == 0)
end

local base = state("alpha", { "2", "3" }, { { "7", "10" } })
local ok, result, trace = observe(base, command("7", "0", "4", "5", false), true)
assert(ok and Seme.result_is_ok(result) and #trace == 1 and trace[1] == true)
local transition = Seme.result_value(result)
local updated = Seme.transition_state(transition)
assert(decimal(Seme.transition_result(transition)) == "24")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 0)) == "14")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 1)) == "3")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 2)) == "14")
assert(decimal(Seme.option_value(Seme.map_lookup(Seme.field(updated, "Counters"), i64("7")))) == "24")
assert(decimal(Seme.index_zero(Seme.field(base, "Values"), 0)) == "2")

ok, result, trace = observe(base, command("7", "1", "4", "5", true), true)
transition = Seme.result_value(result); updated = Seme.transition_state(transition)
assert(ok and #trace == 1 and decimal(Seme.transition_result(transition)) == "35")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 0)) == "2")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 1)) == "25")
assert(decimal(Seme.index_zero(Seme.field(updated, "Values"), 2)) == "25")

assert_error(base, command("8", "0", "1", "1", false), "1")
assert_error(base, command("7", "2", "1", "1", false), "2")
assert_error(state("bad", { "2", "-1", "3" }, { { "7", "10" } }), command("7", "0", "1", "1", false), "3")
ok, result, trace = observe(base, command("7", "0", "1", "1", false), false)
assert(not ok and string.find(result, "seme.capability_absent", 1, true) and #trace == 0)

local corpus_file = assert(io.open(corpus_path, "rb")); local corpus = vim.json.decode(corpus_file:read("*a")); corpus_file:close()
local generated = { success = 0, missing = 0, index = 0, negative = 0, traces = 0, observations = {} }
assert(#corpus == 128)
for _, sequence in ipairs(corpus) do
  local current = state(sequence.initial.Name, sequence.initial.Values, sequence.initial.Counters)
  assert(#sequence.commands == 16)
  for step, item in ipairs(sequence.commands) do
    local generated_command = command(item.Key, item.Index, item.Delta, item.Amount, item.Scale)
    local ran, generated_result, generated_trace = observe(current, generated_command, true)
    assert(ran)
    if Seme.result_is_ok(generated_result) then
      assert(#generated_trace == 1 and generated_trace[1] == true)
      local accepted = Seme.result_value(generated_result)
      current = Seme.transition_state(accepted)
      generated.success, generated.traces = generated.success + 1, generated.traces + 1
      generated.observations[#generated.observations + 1] = { name = sequence.name .. "-" .. step, outcome = "ok", result = decimal(Seme.transition_result(accepted)), trace = { true } }
    else
      local error_code = decimal(Seme.result_value(generated_result))
      assert(#generated_trace == 0 and (error_code == "1" or error_code == "2" or error_code == "3"))
      if error_code == "1" then generated.missing = generated.missing + 1 elseif error_code == "2" then generated.index = generated.index + 1 else generated.negative = generated.negative + 1 end
      generated.observations[#generated.observations + 1] = { name = sequence.name .. "-" .. step, outcome = "error", result = error_code, trace = {} }
    end
  end
end
assert(#generated.observations == 2048 and generated.success + generated.missing + generated.index + generated.negative == 2048 and generated.traces == generated.success)
io.write(vim.json.encode(generated), "\n")
