# Core Execution v10 Freeze Audit

| Property | Evidence |
|---|---|
| Additive evolution | The declaration test proves v10 appends exactly `IntegerSubtract` after every unchanged v9 schema. |
| Ordered semantics | Native and semantic vectors distinguish `left - right` from `right - left`; the Wasm test checks local order before `i64.sub`. |
| Composition | Nested subtraction, multiplication, literals, and parameter reads use one recursive analyzer, emitter, evaluator, and lowerer. |
| Overflow | Native and semantic vectors cover signed i64 modular underflow. |
| Explicit boundary | Division and all other unintroduced operations reject. |
| Frozen predecessors | The v10 gate regenerates v2-v9 construction artifacts byte-for-byte. |
