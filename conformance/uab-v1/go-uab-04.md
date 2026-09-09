# Go UAB-04 evidence

Go UAB-04 composes explicit `[]int64` construction, length and checked index
queries, deterministic traversal, immutable update/append/removal, and immutable
map construction/insert/lookup/removal into one scalar-returning source program.
The source uses `slices.Clone` before `slices.Replace` and `slices.Delete`, and
structurally checked `maps.Clone` closures for map updates and removals, so the
lifted operations preserve value semantics and alias safety.

`scripts/check-go-uab-04.sh` derives every realization from
`fixtures/go-uab-04/program.go` and its single named vector corpus. It compares
the original and projected Go observations, byte-compares the canonical
re-lift, executes the same valid and malformed vectors in the structural
canonical evaluator and standalone Wasm, and executes every named request with
Pulp pinned at `acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`.

The corpus observes update, append, slice removal, traversal, length, indexing,
map insertion, present-key removal, lookup, signed keys, and modular i64
arithmetic. The generic target runtime matrix separately covers missing-key
removal as a successful immutable no-op. Negative and upper indexes, removal bounds, and a short
source slice reject by direct invocation in every realization. Located source
mutations additionally reject a non-unit removal window, unsupported raw map
removal, aliasing map insertion, and multi-element append.

Provider recognition uses resolved standard-library identities and complete AST
shapes; projector, evaluator, and target paths dispatch on canonical schemas,
types, lexical bindings, and call structure. No fixture, package, function, or
vector name selects collection semantics.
