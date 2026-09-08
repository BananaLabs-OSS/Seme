# Core Execution Semantics v25

Core Execution v25 completes the first collection sequence with two neutral,
immutable value operations:

- `CollectionAppend` returns the input collection followed by one new element.
- `CollectionUpdate` returns a collection of the same length with exactly one
  runtime-selected element replaced.

Both operations produce a new logical value. They expose no capacity, backing
array, object identity, allocation strategy, or source-language mutation
mechanism. Update rejects negative indexes and indexes greater than or equal to
the input length.

The bounded Go adapter recognizes `append(values, value)` and the immutable
update idiom `slices.Replace(slices.Clone(values), index, index+1, value)`.
The JavaScript adapter recognizes native copying operations
`values.concat([value])` and `values.with(Number(index), value)`. Their composed
`UpdateAndAppend` implementations lift to byte-identical canonical programs;
native JavaScript projection re-lifts without semantic drift. Mutating
JavaScript `.push` is deliberately unsupported by this immutable profile.

The Pure ABI v2 target returns dynamically sized slice descriptors and packed
i64 payloads. It allocates fresh target storage, copies the source values, then
applies the update or append, so the request payload cannot alias the result.
The target rejects invalid update indexes and append results beyond the bounded
512-element profile. Native Go, native JavaScript, standalone Wasm, and Pulp
agree on the result while preserving the original collection.

This milestone does not claim mutable collection identity, capacity semantics,
sub-slicing, insertion/removal, non-i64 elements, nested collections, or broad
standard-library compatibility. Those require additional neutral semantics or
explicit language/runtime adaptations.
