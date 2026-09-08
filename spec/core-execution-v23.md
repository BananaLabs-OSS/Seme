# Core Execution Semantics v23

Core Execution v23 adds a language-neutral runtime-sized collection type:

- `SliceType` references its element type.
- Length and ordered elements belong to runtime values, not the type.
- Source-language capacity, allocation strategy, mutability, object identity,
  prototypes, and backing storage are not imported into Core.
- Existing `Fold` supplies deterministic left-to-right traversal.

The first adapters map Go `[]int64` and JSDoc-described JavaScript `bigint[]`
to the same `SliceType(i64)`. Idiomatic Go `range` and JavaScript `reduce`
therefore lift to byte-identical canonical programs. JavaScript projection
emits native `bigint[]` and `reduce`, and re-lifts without semantic drift.

The first Wasm/Pulp realization uses Pure ABI v2. Each slice parameter has an
eight-byte little-endian descriptor containing a canonical payload offset and
an element count. Elements are packed signed i64 bit patterns. The decoder
requires contiguous payloads and an exact total request length, bounds each
slice to 512 elements, and rejects requests larger than 7160 bytes before
evaluation. These are target-contract decisions, not Core slice semantics.

The generated helper dynamically loops over the supplied element count. Native
Go, native JavaScript, standalone Wasm, and Pulp agree for empty and differing
runtime lengths, signed values, and modular i64 overflow. Tests reject truncated
headers, noncanonical offsets, absent or trailing payloads, oversized counts,
and oversized requests.

This profile does not yet claim slice construction or return values, mutation,
sub-slicing, capacity semantics, computed indexing, append, nested slices,
non-i64 elements, or arbitrary fold bodies. Those remain explicit future
semantic and realization work.
