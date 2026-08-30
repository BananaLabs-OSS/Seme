# Core Execution v5 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | Every v4 schema and field identity remains unchanged. |
| Declarative generation | pass | The shared generator reproduces frozen v4 byte-for-byte and deterministically emits v5. |
| Canonical branching | pass | `StringIsEmpty` and `Conditional` represent the source guard independently of target control flow. |
| Explicit failure value | pass | The lifted error arm constructs `AdmitError` and `ResultError`; the success arm remains `ResultOk`. |
| Target independence | pass | The new schemas contain no wire, Wasm, Pulp, pointer, or ABI choices. |
| Executable evidence | pass | Both variants execute in Node and Pulp; a changed Go error literal flows through the canonical graph into a newly derived Wasm response. |

This freezes only the minimum conditional profile needed by the quota proof. It
does not claim general Go control-flow lifting, arbitrary expressions, or a
complete error model.
