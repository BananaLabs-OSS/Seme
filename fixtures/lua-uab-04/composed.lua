---@param first seme.i64
---@param second seme.i64
---@param index seme.i64
---@param replacement seme.i64
---@param appended seme.i64
---@param removeIndex seme.i64
---@param initial seme.i64
---@param keepKey seme.i64
---@param removeKey seme.i64
---@return seme.i64
function Evaluate(first, second, index, replacement, appended, removeIndex, initial, keepKey, removeKey)
  return Seme.add(Seme.fold(Seme.slice(first, second), initial, function(accumulator, element) return Seme.add(accumulator, element) end), Seme.add(Seme.length(Seme.slice_remove(Seme.collection_update(Seme.collection_append(Seme.slice(first, second), appended), index, replacement), removeIndex)), Seme.add(Seme.index_zero(Seme.slice_remove(Seme.collection_update(Seme.collection_append(Seme.slice(first, second), appended), index, replacement), removeIndex), index), Seme.lookup_zero(Seme.map_remove(Seme.map_update(Seme.map_update(Seme.empty_map("i64"), keepKey, initial), removeKey, appended), removeKey), keepKey, "i64"))))
end
