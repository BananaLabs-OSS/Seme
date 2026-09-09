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

local function failure(code)
  return Seme.err(Seme.i64(code))
end

---@seme-id 80111111111111111111111111111104
---@param state State
---@param command Command
---@return seme.result<seme.transition<State,seme.i64>,seme.i64>
function Apply(state, command)
  local key = Seme.field(command, "Key")
  local old_counter = Seme.map_lookup(Seme.field(state, "Counters"), key)
  if not Seme.option_is_some(old_counter) then return failure("1") end

  local values = Seme.field(state, "Values")
  local index = Seme.field(command, "Index")
  if not Seme.less_equal(Seme.i64("0"), index) or
      not Seme.less_equal(index, Seme.i64(tostring(Seme.length(values) - 1))) then
    return failure("2")
  end

  local total = Seme.i64("0")
  local accumulate = function(value) total = Seme.add(total, value) end
  for position = 0, Seme.length(values) - 1 do
    local value = Seme.index_zero(values, position)
    if Seme.less_equal(value, Seme.i64("-1")) then return failure("3") end
    accumulate(value)
  end

  local selected = ApplyPolicy(Seme.field(command, "Scale"), Seme.field(command, "Delta"), total)
  local amount = Seme.field(command, "Amount")
  local add_amount = function(value) return Seme.add(value, amount) end
  local adjusted = add_amount(selected)
  local replaced = Seme.collection_update(values, index, adjusted)
  local new_values = Seme.collection_append(replaced, adjusted)
  local new_counter = Seme.add(Seme.option_value(old_counter), adjusted)
  local new_counters = Seme.map_update(Seme.field(state, "Counters"), key, new_counter)
  local new_state = Seme.record("State", { "Name", "Values", "Counters" }, {
    Name = Seme.field(state, "Name"), Values = new_values, Counters = new_counters,
  })
  local transition = Seme.transition(new_state, new_counter)
  Seme.observe(true)
  return Seme.ok(transition)
end
