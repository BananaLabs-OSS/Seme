# Core Execution v9 Freeze Audit

| Property | Evidence |
|---|---|
| Additive evolution | The declaration test proves v9 appends exactly `IntegerMultiply` after every unchanged v8 schema. |
| Composition | Nested multiplication and addition are recursively analyzed, emitted, and evaluated. Wasm instruction lowering is unit-tested separately; this is not yet an end-to-end structured-function Wasm execution claim. |
| Overflow | Differential vectors include signed i64 modular overflow across the native Go and semantic oracle. |
| Target lowering | The generic recursive Wasm integer lowerer emits `i64.mul` from the canonical node and retains recursive cycle/size guards. The legacy complete application builder does not yet consume the v8 structured function body. |
| Explicit boundary | Subtraction and all other unintroduced operations reject. |
| Frozen predecessors | The v9 gate regenerates v2-v8 construction artifacts byte-for-byte. |
