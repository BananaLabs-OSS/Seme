# Go UAB-05 evidence

Go UAB-05 uses one ordinary Go program with an `Adjuster` interface and two
value-receiver implementations. Runtime Boolean selection boxes either an
offset or scale implementation, then invokes the same interface method. The
two implementations produce distinguishable results and include modular i64
overflow observations.

`scripts/check-go-uab-05.sh` derives every realization from
`fixtures/go-uab-05/program.go` and its single named vector corpus. Original
and projected Go execute the same valid and malformed boundary vectors; the
projection re-lifts byte-identically. The structural canonical evaluator,
standalone Wasm, and Pulp pinned at
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001` execute the same named observations.
The audit fails if any Pulp request is skipped or if any realization disagrees.

Empty, short, long, and noncanonical-selector requests reject through direct
native adapter, canonical, Wasm, and Pulp invocation. Located source mutations
reject missing methods and incompatible signatures. Dedicated forged-graph
tests reject concrete/witness/interface mismatches, requirement and method
membership errors, name/arity/parameter/result mismatches, receiver ownership
errors, cycles, and exhausted evaluation budgets.

The provider resolves Go method sets and standard type identities. Projection,
canonical evaluation, and target dispatch use canonical interfaces,
requirements, witnesses, receiver bindings, and method signatures. No fixture,
function, implementation, or vector name selects behavior.
