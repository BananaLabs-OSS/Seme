local Seme = assert(_G.Seme, "Seme adapters are required")

---@param base seme.i64
---@param value seme.i64
---@return seme.i64
function Immutable(base, value)
  return Seme.immutable_closure_run(base, value)
end

---@param start seme.i64
---@param first seme.i64
---@param second seme.i64
---@return seme.i64
function Mutable(start, first, second)
  return Seme.mutable_closure_run(start, first, second)
end
