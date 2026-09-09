# JavaScript UAB-11 cumulative application design

The two-module native fixture implements an immutable command processor over a
state record containing text, a runtime slice, and a runtime-keyed map. Its
outer Result makes missing keys, invalid indexes, and negative values found
inside traversal explicit errors. Successful execution selects a structural
Adjuster implementation (`sum * Delta` or `sum + Delta`), invokes an immutable
callback that captures and adds `Amount` and a mutable summing closure, performs
modular collection updates, emits one authorized Boolean
observation after validation, and returns the new counter in a state transition.

`javascript-uab-11-native-runner.mjs` is the native oracle. A fixed xorshift
seed generates exactly 128 independent sequences of 16 commands. Each step
asserts that the input state was not mutated, errors produce no trace, and a
success produces `[true]` and returns the counter stored in the new state.

The v35 source gate proves two-file lifting, Kernel validation, executable
projection/native parity, byte-identical re-lift, stable entry identity through
rename, located source rejection, and structural graph rejection. The target
gate derives a recursive Pure Value ABI solely from the canonical entry's type
entities and executes the canonical graph in a reusable Wasm reactor. It
compares all 2,048 commands against the native oracle both standalone and
through Pulp pinned to `acc66ca61fe69c5f2c4093bc55e13aeac6dcc001`.

The Wasm cell receives the graph at initialization and the recursive ABI bytes
at each call; it has no JavaScript source, fixture path, package switch, or
application-name switch. Capability denial, malformed Boolean encoding,
truncation, trailing payload, and malformed graph cases reject with no response
or partial observation through standalone Wasm and pinned Pulp. The specialized
native semantic oracle remains only the expected-behavior producer; canonical
execution has no native JavaScript runtime dependency.

Together, `check-javascript-uab-11-native.sh`,
`check-javascript-uab-11-source.sh`, and
`check-javascript-uab-11-target.sh` establish all five UAB evidence classes.
`check-javascript-uab-11.sh` is the complete acceptance gate and reports lift,
native parity, target parity, projection round trip, and rejection explicitly.
The adjacent `reject-raw-index.js` fixture anchors the located rejection for a
JavaScript coercive/raw indexing implementation that cannot be assigned Core's
checked-index meaning.
