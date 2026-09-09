# Go UAB-02 partial-evidence audit

Go UAB-02 is **not certified**. `scripts/check-uab-v1-go-02.sh` checks all
currently available prerequisites, but deliberately reports partial evidence.
It must not be used to award the cell or change the scorecard.

The direct Go projector tests prove lift, native execution, direct projection,
byte-identical re-lift, and rejection. The historical Core gates independently
prove standalone target and pinned Pulp behavior. That aggregation does not
satisfy the frozen cell wording: native Go and canonical execution must produce
the same boundary/adversarial observations, and the target must produce those
same observations.

Exact missing links:

- `i64`, Boolean, and text projection use the UAB-01 multi-call native oracle,
  while cumulative v30 target evidence executes different programs and vectors.
- bytes, `Result`, and `Option` projection executes independent constructors and
  matches. The composite target instead executes a hand-authored
  `Option<Result<bytes,text>> -> bool` canonical program. No Go source is yet
  lifted into that exact target program and run against its none, ok, error,
  malformed-tag, length, and payload vectors.
- records, fixed arrays, slices, and maps use the same broad semantics and often
  the same source fixtures, but the projector-native tests use different inputs
  from the standalone/Pulp gates. Their successful observations therefore have
  not been compared byte-for-byte across all three paths.
- rejection is also split: projector rejection, canonical corruption rejection,
  and target malformed-input rejection are not driven from one per-family
  adversarial vector table.

Certification requires a per-family vector table consumed by the original Go,
the directly projected/re-lifted Go, canonical execution, standalone Wasm, and
pinned Pulp paths, with exact observation comparison and the same malformed or
out-of-bounds cases. For bytes/`Result`/`Option`, the generic target path must
first lower the canonical graph produced from the matching Go source (or the Go
bridge must be shown to produce the existing composite program byte-identically).

`Result` and `Option` use explicit neutral tagged schemas and total matches;
they are not encoded as Go multiple returns, nullable pointers, sentinels, or
one another. Bytes remain opaque bytes rather than text or integer slices.
Projection emits ordinary bounded Go declarations and never uses JavaScript or
another language as an intermediate representation.

The prerequisite target gates archive and build Pulp commit
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`. Absence of that local pinned
checkout is a failed evidence gate, not permission to substitute a moving
revision or skip target parity.
