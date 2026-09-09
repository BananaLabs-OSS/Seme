# Lua UAB-09 semantic design

`Seme.observe(boolean)` is an adapted, sealed mapping to the canonical
`observability.log` effect. It requires an explicitly installed native
capability callback; absence rejects before any observation. Two calls preserve
block order. Arbitrary globals, `print`, errors, and callbacks are not silently
treated as portable effects. The gate proves original/projected Lua, canonical
authorized and denied execution, Wasm, granted/denied pinned Pulp, exact ordered
ledgers, malformed requests, located source rejection, and graph forgery rejection.
