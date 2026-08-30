# Wasm target v1 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Canonical input authority | pass | Backend accepts only the checked executable Target Contract graph and validates the complete supported expression/effect profile. |
| Deterministic artifact | pass | Two lowerings are byte-identical to the pinned 493-byte Wasm reactor and hash. |
| Call preservation | pass | The resolved imported helper lowers as a separate Wasm function invoked by the provider function; lowering does not select it by source name. |
| Operand derivation | pass | The helper's Wasm `local.get` indexes are derived from canonical ParameterRead references; the fixture deliberately uses reversed addition order and comparison direction. |
| Recursive expression lowering | pass | ParameterRead and IntegerAdd are compiled by a bounded recursive graph walk with cycle detection; IntegerLessEqual supplies the checked Boolean root. |
| Integer literals | pass | A canonical typed zero nested in the helper graph lowers to `i64.const 0` and executes through Node and Pulp. |
| Independent runtime | pass | Runtime command uses Node, Wasm, host adapter, and Application Wire v2 requests with no Go component. |
| Behavioral fidelity | pass | ResultOk behavior matches Go over five vectors; empty subject returns ResultError before logging. |
| Boundary fidelity | pass | Wasm import realizes the canonical adapted `seme.pulp.log-v1` Boundary and remains labeled adapted. |
| Schema-derived codec | pass | Request offsets, request length, response offset, and response length are computed from validated Core Execution v3 RecordType/RecordField entities. |
| Policy enforcement | pass | The valid exact-only non-executable plan cannot be lowered. |
| Malformed input | pass | A short Application Wire v2 request returns status 2 without performing an effect. |
| Source-derived payload | pass | Re-ingesting a changed Go error literal produces a distinct Wasm module returning the changed message. |
| Native continuity | pass | The ordinary Go source remains byte-for-byte unchanged and passes its tests. |

This freeze covers the scoped module and Node host conformance adapter. The
subsequent Pulp proof executes the same reactor; general Wasm lowering,
Component Model/WIT support, and WASI behavior remain outside it.
