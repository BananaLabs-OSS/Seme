---@param value seme.i64
---@return seme.i64
function ObserveI64(value)
  return Seme.add(value, Seme.i64_literal("1"))
end

---@param value boolean
---@return boolean
function ObserveBoolean(value)
  return value
end

---@param value seme.text
---@return seme.text
function ObserveText(value)
  return Seme.text_concat(value, "!")
end
