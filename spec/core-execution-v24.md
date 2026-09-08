# Core Execution Semantics v24

Core Execution v24 adds two language-neutral collection queries:

- `CollectionLength` returns the number of ordered elements as canonical i64.
- `DynamicIndexRead` reads the element selected by a runtime i64 index.

Neither construct specifies source syntax, capacity, object properties, pointer
layout, or bounds-check implementation. A target must reject negative indexes
and indexes greater than or equal to the current collection length.

The first adapters map Go `len(values)` and `values[index]`, and JavaScript
`values.length` and `values[index]`, onto these nodes for bounded i64 slices.
The JavaScript adapter also normalizes safe integer literals used in native
length/index expressions. Go and JavaScript implementations of `LastOr` lift
to byte-identical canonical programs; native JavaScript projection re-lifts
without drift.

The Pure ABI v2 realization obtains slice length from the validated runtime
descriptor and lowers computed access with an unsigned bounds check before
loading packed i64 payload data. Consequently negative indexes also reject.
Empty collections take the explicit fallback branch without accessing memory.
Native Go, native JavaScript, standalone Wasm, and Pulp agree on empty,
single-element, and multi-element behavior.

This profile does not yet claim indexing for every collection type, mutation,
sub-slicing, append, slice construction/results, non-i64 elements, or general
numeric conversion semantics. JavaScript Number length/index expressions are
accepted only where their values are safe integers and the target collection
bound makes the canonical i64 mapping exact.
