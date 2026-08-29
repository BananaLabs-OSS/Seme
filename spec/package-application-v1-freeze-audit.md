# Package/Application v1 freeze audit

The first package/application proof is frozen for the exact quota-policy profile
defined by Package Contract v1 and Core Execution v2.

| Property | Result | Evidence |
|---|---|---|
| Ordinary project continuity | pass | The fixture passes `go test ./...` before import and every source byte remains unchanged. |
| Canonical package boundary | pass | Stable Package and TypedInterface entities bind the exported API to the provider Function identity. |
| Complete assumptions | pass | Dependency and effect lists are explicitly empty; modular signed-i64 behavior is an explicit runtime assumption. |
| Mapping fidelity | pass | Exact realization and type-check/differential evidence are canonical fields of the mapping. |
| Independent behavior | pass | A reproducible Seme-owned interpreter matches five Go vectors including overflow. |
| Determinism | pass | Module generation and application lifting reproduce byte-for-byte. |
| Fail-closed behavior | pass | Unsupported source, absent package evidence, and malformed arguments reject. |

The freeze covers the boundary contract and this exact profile. Nonempty effects
and dependencies, records, general calls, package resolution, Wasm planning, and
Pulp execution remain outside it and require their own evidence.
