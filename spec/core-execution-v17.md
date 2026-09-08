# Core Execution Semantics v17

Core Execution v17 makes the record vocabulary introduced in v3 available to
the generic compositional execution path. `RecordType` owns ordered
`RecordField` identities. `RecordConstruct` supplies exactly one value per
field in canonical field order, and `FieldRead` selects by field identity—not
by source spelling or target byte offset.

Ordinary Go structs, keyed composite literals, and selectors are exact source
views for the bounded immutable profile. JavaScript JSDoc object typedefs,
plain object literals, and property reads provide a native structural view.
Both providers converge on identical canonical record and field identities;
JavaScript projection emits normal typedefs and objects and re-lifts without
drift.

The first pure Wasm realization supports records as internal immutable values
whose selected fields reduce to existing scalar and string expressions. It
validates record membership, field order, construction arity, cycles, and
bounded traversal before scalar replacement. Record-valued ABI parameters and
results, nested records, updates, references, and identity-bearing objects are
not claimed by this profile.
