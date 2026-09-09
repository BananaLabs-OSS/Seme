# JavaScript UAB-06 semantic design and evidence

Status: complete. `scripts/check-javascript-uab-06.sh` binds the exact five
evidence classes to the scorecard and executes the original and projected
native JavaScript oracles, byte-identical re-lift, canonical evaluation with
named malformed calls, standalone Wasm, pinned Pulp, located source rejection,
and adversarial canonical closure forgeries.

The bounded exact cell maps lexical JavaScript arrow closures to neutral Core
function types, capture bindings, closure construction, and indirect calls.
Immutable capture stores a value snapshot. Mutable capture stores an isolated
explicit environment whose update is returned as state and committed by the
caller; it is not JavaScript heap aliasing disguised as a neutral value.

One source corpus contains `MakeAdder` and `MakeCounter`. Shared vectors make
the immutable captured base observable, make mutation across two counter calls
observable, distinguish call ordering, and cover modular i64 overflow.

Dynamic `this`, `arguments`, direct or indirect `eval`, async callbacks,
generators, getters, proxies, host objects, and closure sharing through arbitrary
JavaScript aliases are outside this exact cell. They require explicit adapted or
native-island semantics rather than a false exact projection.

Certification remains pending. Canonical evaluation does not yet execute Core
v28/v29 closure schemas, and target backend selection currently examines
unreachable closure entities across the whole graph instead of the selected
entry's reachable subgraph.
