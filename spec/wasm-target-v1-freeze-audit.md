# Wasm target v1 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Canonical input authority | pass | Backend accepts only the checked executable Target Contract graph and validates the complete supported expression/effect profile. |
| Deterministic artifact | pass | Two lowerings are byte-identical to the pinned 205-byte Wasm reactor and hash. |
| Independent runtime | pass | Runtime command uses Node, Wasm, host adapter, and scalar inputs with no Go component. |
| Behavioral fidelity | pass | Result and logging-effect trace match Go over five vectors including modular overflow. |
| Boundary fidelity | pass | Wasm import realizes the canonical adapted `seme.pulp.log-v1` Boundary and remains labeled adapted. |
| Policy enforcement | pass | The valid exact-only non-executable plan cannot be lowered. |
| Native continuity | pass | The ordinary Go source remains byte-for-byte unchanged and passes its tests. |

This freeze covers the scoped module and Node host conformance adapter. The
subsequent Pulp proof executes the same reactor; general Wasm lowering,
Component Model/WIT support, and WASI behavior remain outside it.
