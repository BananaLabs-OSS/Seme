# Core Execution v3 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | All Core Execution v2 schema and field identities remain unchanged. |
| Declarative generation | pass | One shared generator emits v2 and v3; frozen v2 remains byte-identical. |
| Record identity | pass | Record types and fields receive stable canonical identities derived from the lifted declaration. |
| Ordered fields | pass | Every field has an explicit canonical index validated independently by the Wasm backend. |
| Expression coverage | pass | Field reads and record construction are explicit graph entities rather than target-only assumptions. |
| Target independence | pass | The canonical record entities contain no byte offsets, Wasm types, or Pulp ABI data. |

The freeze covers named records containing the current scalar type system.
Optional fields, variants/results, collections, strings, recursion, alignment
policies, and schema evolution remain later additive profiles.
