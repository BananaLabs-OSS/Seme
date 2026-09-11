# JavaScript UPB-04 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-04.sh`.

The cumulative gate proves all seven evidence classes for a typed call across
two native ES modules. `Run` passes `bigint[]` plus six `bigint` values to the
separately owned `Transform` provider. The callee composes checked indexing,
slice construction/update/append/removal, fold, and map insert/lookup/removal.
Package-v2 records the exact canonical `slice<i64> + 6×i64 → i64` signatures,
source origins, public exports, and import binding.

The complete Project-v4 source/dependency chain reproduces, projects back into
two functioning native modules, preserves package metadata, and re-resolves its
pinned offline npm closure. Original and projected native behavior agree.
Every named valid and malformed collection observation agrees across native
JavaScript, canonical execution, standalone Wasm, and pinned Pulp. Raw or
malformed collection mechanics, import/visibility faults, dependency drift,
graph tampering, and publication collisions reject through cumulative gates.

This bounded claim covers the declared collection vocabulary and boundary; it
does not imply arbitrary JavaScript objects, npm APIs, or runtime behavior.
