# JavaScript UAB-02 partial evidence

JavaScript UAB-02 is **not certified**. The bridge covers every required family,
but the historical collection gates do not yet drive identical boundary and
adversarial vectors through original native source, projected native source,
canonical execution, standalone Wasm, and pinned Pulp.

The bounded ECMAScript/Node profile maps JavaScript `bigint` to signed Seme
i64 as an explicit adaptation. Boolean and scalar-valid strings map exactly.
`Uint8Array` maps to opaque bytes. Records, fixed arrays, arrays used as Seme
slices, and `Map<bigint,bigint>` retain distinct canonical types.

JavaScript has no native total Option or Result value. The explicit immutable
`Seme` adapter therefore exposes tagged values and total matches. It does not
reinterpret `null`, `undefined`, exceptions, truthiness, Number, or coercive
equality. Those nearby forms reject in the provider fidelity suite.

`scripts/check-javascript-uab-02.sh` aggregates the existing direct JavaScript
value-family proofs and the JavaScript-derived v32 composite proof. Each source
is parsed by Acorn, runs under Node as the native oracle, projects back to
reviewable JavaScript and re-lifts, and executes through deterministic Wasm and
the pinned Pulp proof runtime. The composite proof specifically executes a
directly lifted `Option<Result<Uint8Array,string>> -> boolean` function rather
than substituting the consumer-neutral fixture.

This aggregate gate records prerequisite coverage only. It must not award the
UAB-02 cell until shared per-family observation tables close the cross-path
vector gaps.
