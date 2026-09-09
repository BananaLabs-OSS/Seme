# JavaScript UAB-05 evidence

JavaScript UAB-05 maps a frozen structural protocol declaration to neutral Core
`InterfaceType`, `MethodRequirement`, `SatisfactionWitness`, `InterfaceValue`,
and `DynamicMethodCall` semantics. JSDoc supplies the closed protocol and method
signatures; declared classes supply immutable records and native methods.
Witnesses are resolved structurally during lifting, never through a mutable
JavaScript prototype lookup at execution time.

`scripts/check-javascript-uab-05.sh` derives every realization from one source
and one four-vector corpus. The vectors select both the offset and scale
witnesses, make the selected implementation observably change the answer, and
exercise signed modular overflow. Original Node, projected Node, the independent
canonical evaluator, standalone Wasm, and Pulp pinned at
`acc66ca61fe69c5f2c4093bc55e13aeac6dcc001` produce the same exact named i64
observations. The projection re-lifts byte-identically.

The standalone Wasm and pinned Pulp executions directly reject named empty,
short, long, and invalid-Boolean-selector requests. Canonical evaluation
directly rejects arguments with wrong selector and value types or arity.
Located source mutations reject prototype replacement and a missing structural
method rather than silently dropping JavaScript behavior. Generic adversarial
evaluator tests additionally forge the witness concrete type, witness interface,
method name, arity, parameter type, result type, and receiver ownership; every
forgery rejects.

Ordinary prototype mutation, getters and setters, proxies, symbols,
`call`/`apply`, detached or rebound `this`, inheritance, and private fields are
not claimed as exact neutral interface semantics. They must remain explicitly
JavaScript-specific or use a future adapted/native-island realization.

Provider, projector, evaluator, and target decisions use canonical schemas,
resolved types, lexical ownership, and witness relationships. No fixture,
package, function, implementation, or vector name selects behavior.
