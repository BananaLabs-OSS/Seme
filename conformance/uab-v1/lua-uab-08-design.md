# Lua UAB-08 semantic design

Lua uses explicit sealed `Seme.ok` and `Seme.err` values and a total adapter
that maps both arms. Errors are propagated unchanged. Exceptions, `nil`, false
sentinels, multiple returns, and truthiness are not treated as canonical Result.
The corpus distinguishes success, error propagation, and i64 wrap. The gate
binds original/projected Lua, byte re-lift, canonical/Wasm/Pulp exact ledgers,
located source rejection, and forged graph rejection.
