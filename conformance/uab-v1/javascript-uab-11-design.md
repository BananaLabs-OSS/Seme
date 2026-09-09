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

The v35 source gate now proves two-file lifting, Kernel validation, executable
projection/native parity, byte-identical re-lift, stable entry identity through
rename, located source rejection, and structural graph rejection. Generic
canonical graph execution, recursive Pure Value ABI execution, pinned Pulp
execution, capability denial, and ABI rejection remain acceptance-gate work.
The specialized native semantic oracle is not canonical evidence. No UAB score
is awarded by this incomplete checkpoint.
The adjacent `reject-raw-index.js` fixture anchors the located rejection for a
JavaScript coercive/raw indexing implementation that cannot be assigned Core's
checked-index meaning.
