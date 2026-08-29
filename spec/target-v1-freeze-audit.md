# Target Contract v1 freeze audit

Target Contract v1 and its first Wasm/Pulp planning profile are frozen at the
canonical planning boundary.

| Property | Result | Evidence |
|---|---|---|
| Canonical vocabulary | pass | Generated G1 and compiled Seme reproduce and pass Kernel/Foundation validation. |
| Native continuity | pass | The unchanged ordinary Go package passes its normal tests and no source byte is projected. |
| Dependency discovery | pass | `go/types` resolves the called function to standard package `log.Printf`. |
| Effect accounting | pass | `observability.log` Effect and Capability entities are explicit package requirements. |
| Honest adaptation | pass | Wasm/Pulp support rules, package mapping, resolutions, and Boundary retain fidelity `adapted`. |
| Policy rejection | pass | Exact-only policy produces `impossible` resolutions and a non-executable plan without relabeling rules. |
| Determinism | pass | Ingestion and both policy plans reproduce byte-for-byte. |
| Fail-closed scope | pass | Stale native evidence, changed message semantics, and unknown policy reject. |

This freeze does not claim that a Wasm artifact exists or that Pulp has executed
the component. It freezes the canonical input that the emitter and runtime must
honor next.
