# JavaScript UAB-02 acceptance evidence

JavaScript UAB-02 has one complete acceptance gate covering every required
family. Independent integration review awarded all five evidence classes.

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

The acceptance gate uses versioned shared vector tables for scalar, collection,
and tagged-composite programs. Original and projected JavaScript, the
structural canonical evaluator, standalone Wasm, and pinned Pulp must agree on
every valid observation. Malformed native values, canonical values, and wire
messages use representation-appropriate cases from the same semantic
categories; they need not use byte-identical encodings across representations.

Checked indexing is an explicit adaptation as well. Raw JavaScript bracket
access can return `undefined` for an invalid index, whereas canonical Seme
rejects the operation. The bounded bridge therefore uses `Seme.array` and
`Seme.index`; their native implementation requires a BigInt index and rejects
negative and upper-bound indices. Raw bracket indexing is rejected by the
provider rather than silently receiving Seme's stricter meaning.
