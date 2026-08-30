# Core Execution v4 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | Every v3 schema and field identity remains unchanged. |
| Declarative generation | pass | The shared generator reproduces frozen v2 and v3 byte-for-byte before emitting v4. |
| Variable-width semantics | pass | String and byte-sequence types are canonical without prescribing target representation. |
| Explicit fallibility | pass | Result type, success construction, and error construction are distinct entities. |
| Target independence | pass | No encoding, pointer, allocator, Wasm, Pulp, or ABI fields occur in the semantic schemas. |

This freeze establishes the type vocabulary. Go lifting, bounded Application
Wire v2 encoding, both result branches, and runtime conformance are subsequent
proof obligations and are not claimed by this audit.
