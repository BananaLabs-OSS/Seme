# Lua UAB-05 semantic design

Lua has no native nominal interface or method-set construct. Ordinary tables,
metatables, `__index`, colon-call receiver insertion, and mutable function
fields carry Lua-specific identity and mutation semantics that Core methods and
explicit satisfaction witnesses do not imply. UAB-05 therefore uses explicit
Seme-owned protocol values rather than claiming raw Lua object equivalence.

The native-equivalent surface is:

```lua
local Adjuster = Seme.protocol("Adjuster", { "adjust" })
local OffsetAdjuster = Seme.implementation(Adjuster, "Offset", {
  adjust = Offset_adjust,
})
local value = Seme.interface_value(OffsetAdjuster, offset)
return Seme.dynamic_call(value, "adjust", input)
```

`Seme.protocol` is closed, ordered, duplicate-free, and bounded to 32
requirements. `Seme.implementation` requires exactly one Lua function for each
declared requirement and rejects missing or extra entries. An interface value
checks the explicit concrete adapter kind, and dynamic calls may select only a
declared requirement. The adapter copies and freezes its protocol and witness
descriptions; later mutation of the source tables cannot change dispatch.

The provider must map this surface structurally to Core v27 `InterfaceType`,
`MethodRequirement`, `SatisfactionWitness`, `InterfaceValue`, and
`DynamicMethodCall`, with the implementation functions mapped to v26 `Method`
and explicit receiver bindings. It must resolve declarations and types
lexically; names alone never establish satisfaction. Projection must reproduce
the explicit adapter surface, not metatable sugar.

The eventual evidence fixture should contain two record implementations of one
`i64 -> i64` requirement, select between their witnesses using a Boolean, and
dispatch over signed and modular-boundary vectors. Rejections must include raw
metatable dispatch, colon-call inference, missing and extra methods, wrong
receiver kinds, forged witnesses, mismatched requirements, duplicate
requirements, and more than 32 implementations or requirements. No UAB-05
score is claimed until native, canonical, projected/re-lifted, standalone Wasm,
and pinned Pulp observations all agree on one shared corpus.
