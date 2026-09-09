local Seme = dofile("reference/lua/seme-values.lua")

local function Check(value)
  return Seme.match_option(value, false, function(some)
    return Seme.match_result(some,
      function(ok) return Seme.bytes_equal(ok, Seme.bytes_literal("ok")) end,
      function(err) return Seme.text_equal(err, "bad") end)
  end)
end

assert(Check(Seme.none()) == false)
assert(Check(Seme.some(Seme.ok(Seme.bytes("ok")))) == true)
assert(Check(Seme.some(Seme.ok(Seme.bytes("no")))) == false)
assert(Check(Seme.some(Seme.err(Seme.text("bad")))) == true)
assert(Check(Seme.some(Seme.err(Seme.text("other")))) == false)
print("Lua composite v32 native vectors: ok")
