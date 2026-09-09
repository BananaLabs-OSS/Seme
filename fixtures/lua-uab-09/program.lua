local Seme = assert(_G.Seme, "Seme adapters are required")

---@param first boolean
---@param second boolean
---@return boolean
function Observe(first, second)
  Seme.observe(first)
  Seme.observe(second)
  return second
end
