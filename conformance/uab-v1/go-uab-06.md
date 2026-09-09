# Go UAB-06 evidence

Status: complete. `scripts/check-go-uab-06.sh` binds the exact five evidence
classes to the scorecard. One Go source contains immutable lexical capture and
a mutable persistent captured environment. The gate executes shared signed,
ordering, overflow, and malformed vectors through original and projected Go,
byte-identical re-lift, the canonical evaluator, standalone Wasm, and pinned
Pulp. It also runs adversarial canonical closure tests and requires a located
source rejection for an unsupported concurrent captured environment.

Go closures are lifted directly to Core closure and environment semantics.
Projection reconstructs idiomatic native Go closures; it does not translate
through JavaScript or Lua.
