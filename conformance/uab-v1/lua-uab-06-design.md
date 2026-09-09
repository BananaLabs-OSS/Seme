# Lua UAB-06 semantic design and evidence

Lua closures normally capture mutable lexical cells and permit aliases, coroutines,
metatable-mediated values, and host interactions. The bounded exact bridge therefore
uses explicit `Seme.immutable_closure_run` and `Seme.mutable_closure_run` adapters.
The former snapshots one i64 capture. The latter owns one isolated mutable i64
environment and explicitly commits the environment returned by its first stateful
call before its second call. Raw Lua closures are not falsely claimed as exact.

One source contains both forms. Shared vectors expose capture values, persistent
mutation, call order, signed values, and modular i64 wrap. The acceptance gate runs
the original and projected Lua, byte-identical re-lift, canonical evaluation with
named malformed invocations, standalone Wasm, pinned Pulp, and located rejection.
