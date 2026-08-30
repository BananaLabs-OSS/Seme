# Wasm target v1 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Canonical input authority | pass | Backend accepts only the checked executable Target Contract graph and validates the complete supported expression/effect profile. |
| Deterministic artifact | pass | Two lowerings are byte-identical to the pinned 471-byte Wasm reactor and hash. |
| Independent runtime | pass | Runtime command uses Node, Wasm, host adapter, and Application Wire v2 requests with no Go component. |
| Behavioral fidelity | pass | ResultOk behavior matches Go over five vectors; empty subject returns ResultError before logging. |
| Boundary fidelity | pass | Wasm import realizes the canonical adapted `seme.pulp.log-v1` Boundary and remains labeled adapted. |
| Schema-derived codec | pass | Request offsets, request length, response offset, and response length are computed from validated Core Execution v3 RecordType/RecordField entities. |
| Policy enforcement | pass | The valid exact-only non-executable plan cannot be lowered. |
| Malformed input | pass | A short Application Wire v2 request returns status 2 without performing an effect. |
| Native continuity | pass | The ordinary Go source remains byte-for-byte unchanged and passes its tests. |

This freeze covers the scoped module and Node host conformance adapter. The
subsequent Pulp proof executes the same reactor; general Wasm lowering,
Component Model/WIT support, and WASI behavior remain outside it.
