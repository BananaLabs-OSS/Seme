# JavaScript fidelity boundary for UAB-v1

The JavaScript bridge targets ECMAScript 2024 source parsed as an ECMAScript
module and executed by the pinned Node environment used by the acceptance
gate. Canonical signed `i64` values project to JavaScript `bigint`; this is an
explicit **adaptation**, not a claim that JavaScript `number` has i64 behavior.

Canonical text contains Unicode scalar values. JavaScript strings use UTF-16,
so valid scalar text is adapted losslessly while lone surrogate code units are
rejected. The provider must not silently assign Core meaning to JavaScript
coercion, truthiness, `null`, `undefined`, exceptions, promises, generators,
prototypes, or other host-runtime behavior.

`javascript-uab-v1-fidelity.test.mjs` freezes executable rejection evidence for
the currently exposed boundary: Number-versus-BigInt, i64 range, coercive
equality, non-Boolean truthiness, nullish values, exceptions, async functions,
generators, and lone surrogates. Supporting one of these later requires an
explicit semantic schema and fidelity classification; weakening the rejection
is not sufficient.

This boundary evidence is necessary but does not independently pass a UAB-v1
cell. A cell is certified only when its lift, native parity, target parity,
projection round trip, and rejection evidence are connected by the profile's
acceptance gate.
