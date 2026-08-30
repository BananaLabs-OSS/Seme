# Wasm/Pulp v1 target artifact

`quota-admit.wasm` is the deterministic 471-byte Pulp reactor derived from the
checked `allow-adapted` Target Contract v1 plan. It imports
`pulp.log_bool(i32) -> i32`, exports `pulp_on_call`, and implements the Pulp
cell lifecycle exports and memory contract without a Go runtime. Core Execution
v4 supplies canonical request/response records, string/bytes fields, and a
Result type. Application Wire v2 derives three little-endian signed i64 fields,
bounded UTF-8 subject and evidence bytes, an `Ok` response preserving all
fields, and an Error response carrying the lifted Go error message.
`scripts/check-wasm-v1.sh` reproduces and executes it.

The Node conformance runner checks five success vectors, one error vector, and
malformed input. The two
cell manifests additionally prove repeated granted calls and denied execution
in actual Pulp. `scripts/demo-application-v1.sh` presents the complete slice.
