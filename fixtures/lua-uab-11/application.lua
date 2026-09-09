local Seme = assert(_G.Seme, "Seme adapters are required")

---@class State
---@field Name seme.text
---@field Values seme.slice<seme.i64>
---@field Counters seme.map<seme.i64,seme.i64>

---@class Command
---@field Key seme.i64
---@field Index seme.i64
---@field Delta seme.i64
---@field Amount seme.i64
---@field Scale boolean

---@param code seme.i64
---@return seme.result<seme.transition<State,seme.i64>,seme.i64>
local function failure(code)
  return Seme.err(code)
end

---@seme-id 80111111111111111111111111111104
---@param state State
---@param command Command
---@return seme.result<seme.transition<State,seme.i64>,seme.i64>
function Apply(state, command)
  local key = Seme.field(command, "Key")
  local old_counter = Seme.map_lookup(Seme.field(state, "Counters"), key)
  return Seme.match_option(old_counter, Seme.err(Seme.i64_literal("1")), function(counter)

  local values = Seme.field(state, "Values")
  local index = Seme.field(command, "Index")
  if Seme.less_equal(index, Seme.i64_literal("-1")) then
    return Seme.err(Seme.i64_literal("2"))
  end
  if Seme.less_equal(Seme.length_i64(values), index) then
    return Seme.err(Seme.i64_literal("2"))
  end

  local total = Seme.i64_literal("0")
  local accumulate = function(value) total = Seme.add(total, value) end
  local position = Seme.i64_literal("0")
  while Seme.less_equal(position, Seme.add(Seme.length_i64(values), Seme.i64_literal("-1"))) do
    local value = Seme.index_zero(values, position)
    if Seme.less_equal(value, Seme.i64_literal("-1")) then
      return Seme.err(Seme.i64_literal("3"))
    end
    accumulate(value)
    position = Seme.add(position, Seme.i64_literal("1"))
  end

  local selected = ApplyPolicy(Seme.field(command, "Scale"), Seme.field(command, "Delta"), total)
  local amount = Seme.field(command, "Amount")
  local add_amount = function(value) return Seme.add(value, amount) end
  local adjusted = add_amount(selected)
  local replaced = Seme.collection_update(values, index, adjusted)
  local new_values = Seme.collection_append(replaced, adjusted)
  local new_counter = Seme.add(counter, adjusted)
  local new_counters = Seme.map_update(Seme.field(state, "Counters"), key, new_counter)
  local new_state = Seme.record("State", { "Name", "Values", "Counters" }, { Name = Seme.field(state, "Name"), Values = new_values, Counters = new_counters })
  local transition = Seme.transition(new_state, new_counter)
  Seme.observe(true)
  return Seme.ok(transition)
  end)
end
