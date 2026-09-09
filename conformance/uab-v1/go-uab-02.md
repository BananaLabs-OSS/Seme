# Go UAB-02 evidence

`scripts/check-uab-v1-go-02.sh` is the complete evidence gate for the bounded
Go UAB-02 cell. It proves signed i64, Boolean, text, opaque bytes, records,
`Result`, `Option`, fixed arrays, slices, and runtime-keyed maps. It does not
claim general Go, standard-library, generic, or collection compatibility.

The scalar gate drives one source-derived program through original Go, direct
canonical lift and independent structural evaluation, direct Go projection, byte-identical re-lift, standalone Wasm,
and Pulp pinned at `acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`. Its shared
observations include i64 zero and modular overflow, both Boolean values, empty
text, and exact Unicode text.

The composite gate does the same, including distinct canonical evaluation, for a directly lifted
`Option<Result<bytes,text>> -> bool` Go program. Original and projected Go,
standalone Wasm, and pinned Pulp agree on none, matching and nonmatching opaque
bytes, and matching and nonmatching error text. Eight malformed tag, inactive
payload, descriptor, length, contiguity, trailing-data, and UTF-8 vectors reject
through both standalone and Pulp boundaries. `Result` and `Option` remain
explicit neutral tagged schemas; bytes are never replaced by text or i64
slices.

Records, arrays, slices, and maps share one direct Go aggregate fixture that is
lifted, structurally evaluated as canonical Seme, projected/re-lifted, and run
through standalone Wasm and pinned Pulp. Historical gates supply additional
isolated rejection evidence. The shared observations
cover Unicode record text, every valid array index and both bounds, empty,
signed, variable, modular, maximum-sized and malformed slices, and repeated,
negative, absent, reordered, empty, maximum-sized and malformed maps.

Fail-closed evidence additionally corrupts record order, array/slice/map types
and bounds, Option and match identities, Program call membership, and
unsupported canonical expressions. No language intermediary participates in
Go projection.
