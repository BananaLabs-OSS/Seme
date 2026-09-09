---@param limit seme.i64
---@param enabled boolean
---@param values seme.slice<seme.i64>
---@param probe seme.i64
---@param bypass boolean
---@return seme.i64
function Accumulate(limit, enabled, values, probe, bypass)
  if Seme.boolean(enabled) and Seme.less_equal(Seme.index_zero(values, probe), limit) then
    return Seme.i64_literal("11")
  end
  if Seme.boolean(bypass) or Seme.less_equal(Seme.index_zero(values, probe), limit) then
    return Seme.i64_literal("12")
  end
  local total = Seme.i64_literal("0")
  local index = Seme.i64_literal("0")
  while Seme.boolean(enabled) and Seme.less_equal(index, limit) do
    total = Seme.add(total, index)
    if Seme.equal_i64(index, Seme.i64_literal("3")) then
      return total
    end
    index = Seme.add(index, Seme.i64_literal("1"))
  end
  return total
end
