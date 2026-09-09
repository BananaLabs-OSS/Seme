local function Identity(value)
  return value
end

local function Run(enabled)
  return Identity(enabled)
end

assert(Run(true) == true)
assert(Run(false) == false)
print("lua native UAB-01 oracle: ok")
