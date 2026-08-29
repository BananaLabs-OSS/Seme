# Wasm/Pulp v1 target artifact

`quota-admit.wasm` is the deterministic 205-byte Pulp reactor derived from the
checked `allow-adapted` Target Contract v1 plan. It imports
`pulp.log_bool(i32) -> i32`, exports `admit(i64, i64, i64) -> i32`, and implements the
Pulp cell lifecycle exports and memory contract without a Go runtime.
`scripts/check-wasm-v1.sh` reproduces and executes it.

The Node conformance runner checks five direct result/effect vectors. The two
cell manifests additionally prove granted and denied execution in actual Pulp.
