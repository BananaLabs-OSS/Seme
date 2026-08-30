# Core Execution v6 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | Every v5 schema and field identity remains unchanged. |
| Canonical call edge | pass | `FunctionCall` references a stable `Function` identity and ordered semantic arguments. |
| Provider independence | pass | Go ingestion resolves `Admit` and `WithinLimit` from separate native files and emits distinct Functions rather than copying helper arithmetic into the caller. |
| Backend validation | pass | Wasm lowering validates the callee signature, body, call target, and argument order from the graph. |
| Separate execution | pass | The generated reactor contains a separate Wasm helper function and `pulp_on_call` invokes it with `call`. |
| Runtime continuity | pass | Node and Pulp execute the new artifact with unchanged Result and capability behavior. |

This freezes direct calls in the bounded profile. Recursion, indirect calls,
closures, generics, methods, and effect-polymorphic calls remain future work.
