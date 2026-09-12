local Seme = assert(_G.Seme, "Seme adapters are required")

---@seme-id 80444444444444444444444444444401
---@param values seme.slice<seme.i64>
---@param index seme.i64
---@param replacement seme.i64
---@return seme.i64
local function ReplaceAndRead(values, index, replacement)
  return Seme.index_zero(Seme.collection_update(values, index, replacement), index)
end

return { ReplaceAndRead = ReplaceAndRead }
