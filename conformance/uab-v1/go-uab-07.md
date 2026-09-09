# Go UAB-07 evidence

Status: complete. `scripts/check-go-uab-07.sh` binds the exact five evidence
classes to the scorecard. A typed Go transition returns a new counter state and
the distinct prior value without mutating its input. Original/projected Go,
byte-identical re-lift, canonical Seme, standalone Wasm, and pinned Pulp share
one signed and overflow corpus. The gate also executes exact malformed ledgers,
a located source type error, and wrong-type, swapped-field, and cyclic graph
forgeries.
