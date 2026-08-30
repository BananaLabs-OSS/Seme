# Core Execution v6 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | Every v5 schema and field identity remains unchanged. |
| Canonical call edge | pass | `FunctionCall` references a stable `Function` identity and ordered semantic arguments. |
| Provider independence | pass | Go ingestion resolves a renamed imported helper and emits distinct Functions without selecting the callee by source name or copying arithmetic into the caller. |
| Shared expression analysis | pass | Core v1/v2 lifts and application planning consume the same recursively analyzed ParameterRead/IntegerAdd/IntegerLessEqual provider tree. |
| Backend validation | pass | Wasm lowering validates the callee signature, body, call target, argument order, and resolved helper operand identities from the graph. |
| Presentation normalization | pass | `limit >= delta+current` is normalized to the canonical less-or-equal family while preserving resolved addition operand identities. |
| Separate execution | pass | The generated reactor contains a separate Wasm helper function and `pulp_on_call` invokes it with `call`. |
| Runtime continuity | pass | Node and Pulp execute the new artifact with unchanged Result and capability behavior. |

This freezes direct calls in the bounded profile. Recursion, indirect calls,
closures, generics, methods, and effect-polymorphic calls remain future work.
