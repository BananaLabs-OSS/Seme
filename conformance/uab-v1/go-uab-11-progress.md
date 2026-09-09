# Go UAB-11 complete evidence

This document records the completed Go cumulative application cell.

The ordinary Go fixture in `fixtures/go-uab-11` supplies three native packages
(`model`, `policy`, and `application`) and the frozen cumulative behavior. Its
tests cover validation order, atomic rejection, interface dispatch, immutable
and mutable closures, wrapping i64 arithmetic, immutable collection updates,
the authorized log point, and the shared deterministic 128 by 16 corpus.

Shared canonical infrastructure now includes:

- Core Execution v35 `MapLookupOption` (`a044`, fields `a0440..a0442`);
- generic canonical evaluation of present-zero versus absent map entries;
- generic reached-effect recording with whole-graph authority preflight;
- recursive transition layouts; and
- deterministic recursive encoding/decoding of the cumulative boundary value.

`IncrementalSession` now groups and type-checks source packages through a local
module importer, emits one unified call graph, preserves semantic provenance,
and handles the fixture's aggregate records, instantiated bridge types,
comma-ok map lookup, closures, control flow, transitions, and effects. The
projected ordinary Go program compiles, executes the native corpus, and
re-lifts byte-identically.

`scripts/check-go-uab-11.sh` proves all five evidence classes over 128
independent sequences of 16 stateful commands. The same 2,048 observations
agree through native and projected Go, generic canonical evaluation,
standalone Wasm, and pinned Pulp. Capability denial and malformed source,
canonical graph, and recursive ABI inputs reject atomically.
