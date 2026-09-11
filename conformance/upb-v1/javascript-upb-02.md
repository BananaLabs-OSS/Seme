# JavaScript UPB-02 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-02.sh`.

The cumulative gate proves all seven evidence classes for two real ES modules.
Acorn-derived evidence preserves application and math ownership, exact source
origins, the `./math/sum.js` import and `Sum` alias, public `Run` and `Sum`, and
package-private `normalize`. Callable signatures are recovered from the
canonical graph rather than guessed from native spelling.

Project-v1, Inventory-v2, Package-v2, and Project-v3 artifacts reproduce
byte-for-byte across independent builds. Missing/private imports, cycles,
escapes, malformed graphs, and collisions reject. A Package-v2-aware projector
recreates two ordinary native modules with the authenticated import/export
boundary. Native behavior matches the original and the projected modules
re-lift to identical Execution and Project-v1 semantics. Repeated projected
lifts produce identical Project-v3 authority while honestly retaining their
new source digests. Native, canonical, standalone Wasm, and pinned Pulp results
agree for this exact project.

This bounded claim supports static named relative ES-module imports. Bare npm
dependencies, default/namespace/dynamic imports, re-exports, CommonJS, and
package export maps remain outside UPB-02.
