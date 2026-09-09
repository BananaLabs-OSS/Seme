# JavaScript UAB-10 candidate design

This bounded cell proves that formatting is not semantic identity and that a
provider can recover one declaration identity across an explicit native rename.
The reconciliation evidence is a versioned sidecar, never a JavaScript
annotation. It binds the package, old declaration name, current declaration
name, and independently derivable prior identity.

The original and substantially reformatted sources lift byte-identically. The
renamed source retains the original function identity, projects as ordinary
JavaScript, and re-imports byte-identically when given the same evidence.
Original, renamed, and projected JavaScript agree with canonical, standalone
Wasm, and pinned Pulp execution over zero, positive, negative, and i64-boundary
values.

The provider rejects wrong packages, missing or colliding declarations,
ambiguous mappings, and forged identity bytes. This is a bounded declaration
rename proof; it does not claim general history inference or automatic
reconciliation.

This file documents candidate evidence only. It does not itself award or alter
the UAB score.
