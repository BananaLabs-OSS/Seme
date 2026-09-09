---@class Bounds
---@field bias seme.i64
---@param bounds Bounds
---@param fixed seme.array<seme.i64,2>
---@param dynamic seme.slice<seme.i64>
---@param counts seme.map<seme.i64,seme.i64>
---@param key seme.i64
---@return seme.i64
function Observe(bounds, fixed, dynamic, counts, key)
  return Seme.add(Seme.add(Seme.field(bounds, "bias"), Seme.index_zero(fixed, 1)), Seme.add(Seme.length(dynamic), Seme.lookup_zero(counts, key, "i64")))
end
