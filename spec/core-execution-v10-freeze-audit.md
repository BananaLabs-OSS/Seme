# Core Execution v10 Freeze Audit

| Property | Evidence |
|---|---|
| Additive evolution | The declaration test proves v10 appends exactly `IntegerSubtract` after every unchanged v9 schema. |
| Ordered semantics | Native and semantic vectors distinguish `left - right` from `right - left`; the Wasm test checks local order before `i64.sub`. |
| Composition | Nested subtraction, multiplication, literals, and parameter reads use one recursive analyzer, emitter, and evaluator. Wasm instruction lowering is unit-tested separately; this is not yet an end-to-end structured-function Wasm execution claim. |
| Overflow | Native and semantic vectors cover signed i64 modular underflow. |
| Explicit boundary | Division and all other unintroduced operations reject. |
| Frozen predecessors | The v10 gate regenerates v2-v9 construction artifacts byte-for-byte. |

The legacy complete application builder does not yet consume the v8 structured
function body. This audit therefore freezes the canonical and bounded lowering
evidence without claiming standalone Wasm or Pulp execution.
