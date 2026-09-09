# Lua UAB-03 semantic design and evidence

Lua UAB-03 is certified for this bounded direct-Lua profile. Lua is never
translated through Go or JavaScript.

## Composed proof shape

The first fixture is one ordinary Lua 5.1 function over annotated canonical
`seme.i64` values and a native Lua Boolean:

```lua
local total = Seme.i64_literal("0")
local index = Seme.i64_literal("0")
while enabled and Seme.less_equal(index, limit) do
  total = Seme.add(total, index)
  if Seme.equal_i64(index, Seme.i64_literal("3")) then return total end
  index = Seme.add(index, Seme.i64_literal("1"))
end
return total
```

This single behavior exercises ordered local initialization, reassignment,
modular arithmetic, signed comparison, native left-to-right short-circuit
`and`, a conditional, a loop, and an early return from inside the loop. The
provider recognizes syntax and resolved lexical identities, never the package,
function, or variable names.

## Fidelity boundaries

- `local` and assignment map exactly to canonical lexical bindings and mutable
  places. Reads are resolved by lexical identity, not spelling alone.
- `Seme.add`, `Seme.less_equal`, and `Seme.equal_i64` are explicit exact
  adapters for canonical signed modular i64. Lua 5.1/LuaJIT's ordinary number
  operators are not accepted for `seme.i64`; they have different precision,
  coercion, NaN, infinity, and overflow semantics.
- `and` is accepted only when both operands are statically Boolean. Ordinary
  Lua `and` returns an operand and treats every value except `false` and `nil`
  as truthy, so using it with text, numbers, tables, or tagged values rejects.
- `if` and `while` conditions must be statically Boolean. The bridge never
  silently applies Lua truthiness to a canonical value.
- The bounded loop has no `break`, `continue`, numeric/generic `for`, closure
  capture, metatable interaction, or mutation outside its declared locals.
  Canonical and target evaluation enforce an explicit iteration budget.
- Early return is exact only from the function or its structurally nested
  conditional/loop blocks. Multiple Lua returns and implicit `nil` returns are
  outside this profile.

## Evidence

The complete gate runs the same zero, disabled, below-cutoff, cutoff,
above-cutoff, negative-limit, and i64-boundary vectors through original Lua,
the language-neutral canonical evaluator, projected Lua, deterministic Wasm,
and pinned Pulp. It must also reject unannotated arithmetic, non-Boolean
truthiness, undeclared/forward locals, incompatible assignment, a loop without
a statically bounded profile, multiple return values, and malformed/cyclic
canonical graphs.

The final fixture also composes runtime-checked `slice<i64>` indexing into both
Boolean operators. False-AND and true-OR skip an out-of-range read; true-AND
and false-OR force that read and reject through native Lua, canonical
evaluation, standalone Wasm, and pinned Pulp. One versioned vector corpus
generates every canonical argument and target request. Independent integration
review awarded all five evidence classes.
