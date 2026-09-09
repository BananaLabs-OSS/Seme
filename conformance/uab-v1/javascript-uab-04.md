# JavaScript UAB-04 evidence

JavaScript UAB-04 composes explicit immutable collection construction,
length, checked indexing, traversal, update, append, removal, map lookup,
insertion, and removal in one source function. `Seme.slice`, `Seme.update`,
`Seme.append`, `Seme.remove`, `Seme.emptyMap`, `Seme.mapInsert`,
`Seme.mapRemove`, and `Seme.mapLookupZero` are bounded adapters for canonical
value semantics. They do not reinterpret JavaScript Array mutation, sparse
arrays, property indexing, `Map` identity, coercion, or missing-value behavior.

`scripts/check-javascript-uab-04.sh` derives every realization from the same
source and vector corpus. Original Node source, projected and re-lifted Node
source, the structural canonical evaluator, standalone Wasm, and Pulp pinned
at `acc66ca61fe69c5f2c4093bc55e13aeac6dcc001` produce the same exact named
i64 observations. Projection re-lifts byte-identically.

The corpus includes construction through Core v34 `SliceConstruct`, chained
update/append/`SliceRemove`, deterministic traversal with modular addition,
checked length/index queries, and symbolic immutable map insert/remove/lookup.
Native inputs remain unchanged. Negative and upper indexes, removal bounds,
and a non-slice collection reject by direct invocation in the native,
canonical, standalone Wasm, and Pulp realizations. The Pulp audit retains each
malformed vector name and fails if any request is skipped or accepted. Located
source mutations additionally reject index coercion, unknown removal/lookup
operations, and unwrapped BigInt arithmetic.

Generic provider, projector, evaluator, and target paths dispatch on canonical
schema and resolved lexical/type structure. No fixture, package, function, or
vector name selects the implementation.
