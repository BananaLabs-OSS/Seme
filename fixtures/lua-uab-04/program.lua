---@param first seme.i64
---@param second seme.i64
---@return seme.slice<seme.i64>
function ConstructSlice(first, second)
  return Seme.slice(first, second)
end

---@param values seme.slice<seme.i64>
---@param value seme.i64
---@return seme.slice<seme.i64>
function Append(values, value)
  return Seme.collection_append(values, value)
end

---@param values seme.slice<seme.i64>
---@param index seme.i64
---@param value seme.i64
---@return seme.slice<seme.i64>
function Update(values, index, value)
  return Seme.collection_update(values, index, value)
end

---@param values seme.slice<seme.i64>
---@param index seme.i64
---@return seme.slice<seme.i64>
function Remove(values, index)
  return Seme.slice_remove(values, index)
end

---@param values seme.slice<seme.i64>
---@return seme.i64
function Length(values)
  return Seme.length(values)
end

---@param values seme.slice<seme.i64>
---@param index seme.i64
---@return seme.i64
function Index(values, index)
  return Seme.index_zero(values, index)
end

---@param values seme.slice<seme.i64>
---@param initial seme.i64
---@return seme.i64
function Traverse(values, initial)
  return Seme.fold(values, initial, function(accumulator, element) return Seme.add(accumulator, element) end)
end

---@return seme.map<seme.i64,seme.i64>
function EmptyMap()
  return Seme.empty_map("i64")
end

---@param values seme.map<seme.i64,seme.i64>
---@param key seme.i64
---@param value seme.i64
---@return seme.map<seme.i64,seme.i64>
function Insert(values, key, value)
  return Seme.map_update(values, key, value)
end

---@param values seme.map<seme.i64,seme.i64>
---@param key seme.i64
---@return seme.i64
function Lookup(values, key)
  return Seme.lookup_zero(values, key, "i64")
end

---@param values seme.map<seme.i64,seme.i64>
---@param key seme.i64
---@return seme.map<seme.i64,seme.i64>
function RemoveMap(values, key)
  return Seme.map_remove(values, key)
end
