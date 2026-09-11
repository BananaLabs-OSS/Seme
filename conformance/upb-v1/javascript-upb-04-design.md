# JavaScript UPB-04 acceptance design

Status: accepted implementation design; unclaimed.

JavaScript UPB-04 extends the cumulative Project-v4 fixture with typed behavior
across a real ES-module boundary. The application entry accepts a canonical
`slice<i64>` plus six `i64` values and calls an exported collection-policy
function in another module. That function composes checked index access,
slice construction/update/append/removal, fold, and map insertion/lookup/removal.

The JavaScript adapter owns JSDoc type recovery, ES-module import/export rules,
BigInt representation, and `Seme.*` native support calls. Canonical slice, map,
call, result, and trap meaning remain language-neutral. Package-v2 records the
exact callable signatures and source ownership; Project-v3/Project-v4 retain
the source and dependency authorities inherited from earlier cells.

The gate must prove original and projected native module behavior, exact
canonical re-lift, native/canonical/standalone-Wasm/pinned-Pulp parity, stable
module ownership, and deterministic cumulative publication. Raw JavaScript
indexing, missing/private imports, type disagreement across the call, malformed
boundary values, graph tampering, dependency drift, and output collisions must
reject atomically.

Only `scripts/check-javascript-upb-04.sh` may claim this cell, and it must run
the complete mapped UPB-03 gate before its own seven evidence classes.
