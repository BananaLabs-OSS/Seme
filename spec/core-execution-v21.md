# Core Execution Semantics v21

Core Execution v21 adds three language-neutral schemas:

- `FixedArrayType` references an element type and declares a nonzero length.
- `FixedArrayConstruct` references that type and supplies ordered values.
- `IndexRead` references a collection value and an integer index expression.

The bounded first realization supports fixed arrays of 1–32 signed i64 values
constructed inside an expression. Go `[3]int64{a, b, c}[index]` and JavaScript
`[a, b, c][index]` lift to byte-identical canonical programs. Projection emits
ordinary JavaScript syntax and re-lifts without semantic drift.

The Wasm target validates element type, declared length, actual value count,
index type, graph membership, and expression bounds before emission. Valid
indexes return the selected value. Negative and upper-bound indexes trap both
standalone and through Pulp. Native Go reports invalid indexing with a panic;
JavaScript's bracket operator produces `undefined`. Core therefore treats an
out-of-range read as invalid and leaves the host failure mechanism to the
realization rather than falsely claiming identical exception mechanics.

The boundary extension also accepts Go `[N]int64` parameters and JavaScript
`bigint[N]` JSDoc contracts for lengths 1–32. The Wasm ABI derives a packed
`N * 8` byte field directly from the canonical `FixedArrayType`; generated code
receives a bounded pointer into the validated request and performs typed dynamic
loads. Exact-size requests and first/last indexes pass standalone and through
Pulp. Truncated requests return the established malformed-request status, and
negative or upper-bound indexes trap.

This slice does not yet support array-valued results, array locals, mutation,
nested arrays, Boolean/string/record elements, slices, or iteration. Those
remain explicit follow-on work.
