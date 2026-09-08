# Core Execution Semantics v20

Core Execution v20 adds ordered `EffectInvoke` statements. An invocation
references a Foundation `Effect` identity and an ordered argument list. Its
position in a `Block` defines sequencing relative to other statements. Core
does not duplicate effect names, capabilities, host imports, or target policy.
Providers intern each shared Foundation effect/capability declaration once per
program while emitting a distinct invocation identity for every ordered use.

The first bounded adapters map Go `log.Print(bool)` and JavaScript
`console.log(bool)` to the same `observability.log` effect. This mapping is
reported as `adapted`, not exact: the native APIs have presentation and runtime
behavior beyond the portable observation contract. JavaScript projection emits
ordinary `console.log` and re-lifts without semantic drift.

The certified Wasm realization accepts a Boolean argument and result with one
or more ordered observation invocations. It records
`observability.log` as a required capability, imports the guarded
`pulp.log_bool` adapter, and treats any nonzero host status as fatal. The
granted proof emits `true` then `false` and returns `false`. The identical
artifact under a denied Pulp manifest traps and emits no observation.

This profile does not yet claim effect return values, strings or structured
effect arguments, asynchronous effects, recovery after denial, effects inside
branches/loops/callees, or general standard-library and browser API mapping.
