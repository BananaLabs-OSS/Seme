# Go UAB-03 evidence

`scripts/check-go-uab-03.sh` is the complete bounded Go control-flow gate. One
versioned corpus drives original Go, direct canonical lift and structural
evaluation, projected Go, standalone Wasm, and pinned Pulp. Named observations
are normalized and compared exactly; projection re-lifts byte-identically.

The fixture composes immutable locals, mutable places and assignment, modular
i64 arithmetic, signed comparison, Boolean AND/OR, conditionals, a bounded
loop, and early return. Runtime slice indexing makes short-circuit order
observable: false-AND and true-OR skip an invalid read, while true-AND and
false-OR force it and reject in every realization. Boundary vectors include
i64 minimum, maximum, and modular wrap.

Malformed native arguments, canonical values and graphs, Wasm/Pulp ABI
messages, and an unsupported Go `goto` reject. Provider, projector, evaluator,
and target dispatch use parsed syntax, types, canonical schemas, and lexical
identity rather than package, fixture, function, or variable names.
