local Seme = assert(_G.Seme, "Seme adapters are required")

---@class Counter
---@field Value seme.i64

---@param counter Counter
---@param delta seme.i64
---@return seme.transition<Counter,seme.i64>
function Step(counter, delta)
  return Seme.transition_step(counter, delta, "Value")
end
