# Wasm/Pulp v1 target artifact

`quota-admit.wasm` is the deterministic 78-byte artifact derived from the
checked `allow-adapted` Target Contract v1 plan. It imports
`pulp.log_bool(i32)`, exports `admit(i64, i64, i64) -> i32`, and contains no Go
runtime. `scripts/check-wasm-v1.sh` reproduces and executes it.

The current conformance runner implements the declared host import in Node. The
import contract is shaped for Pulp, but this artifact has not yet run inside the
actual Pulp runtime.
