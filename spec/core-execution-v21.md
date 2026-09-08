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

This slice does not yet support array-valued parameters/results, array locals,
mutation, nested arrays, Boolean/string/record elements, slices, iteration, or
collection values crossing the Wasm ABI. Those remain explicit follow-on work.
