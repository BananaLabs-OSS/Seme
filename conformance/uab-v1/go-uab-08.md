# Go UAB-08 evidence

Status: complete. `scripts/check-go-uab-08.sh` binds all five evidence classes
to the scorecard. Native Go composes a declared `Result[int64,int64]` producer
with a total matcher that increments success values and propagates the error
payload unchanged. Original/projected Go, byte-identical re-lift, canonical
Seme, standalone Wasm, and pinned Pulp share exact success, failure, overflow,
and malformed ledgers. Located source rejection and constructor/call graph
forgeries reject incompatible meaning.
