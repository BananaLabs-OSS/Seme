# Lua UAB-07 semantic design and evidence

The bounded Lua bridge uses a sealed `Seme.transition_step` adapter. It returns
an immutable transition whose `state` is a new typed record and whose `result`
is the prior value. Lua tables are not treated as canonical transitions: their
aliasing and mutation semantics remain outside this exact cell.

The corpus distinguishes state from result and covers signed and modular i64
updates. Its gate executes original/projected Lua, byte-identical re-lift,
canonical evaluation with named malformed inputs, standalone Wasm, pinned Pulp
with an exact ledger, located raw-table rejection, and forged graph rejection.
