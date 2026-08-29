# Wasm/Pulp v1 target artifact

`quota-admit.wasm` is the deterministic 262-byte Pulp reactor derived from the
checked `allow-adapted` Target Contract v1 plan. It imports
`pulp.log_bool(i32) -> i32`, exports `pulp_on_call`, and implements the Pulp
cell lifecycle exports and memory contract without a Go runtime. Application
Wire v1 requests are three little-endian signed i64 fields (`current`, `delta`,
`limit`); the response is one canonical Boolean byte.
`scripts/check-wasm-v1.sh` reproduces and executes it.

The Node conformance runner checks five request/result/effect vectors. The two
cell manifests additionally prove repeated granted calls and denied execution
in actual Pulp. `scripts/demo-application-v1.sh` presents the complete slice.
