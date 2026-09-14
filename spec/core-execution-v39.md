# Core Execution v39: typed native invocation

Core Execution v39 adds `NativeInvocation`, an explicit realization boundary
inside an otherwise canonical function. It records the implementation language,
callable identity, native signature, ordered canonical arguments, and canonical
result type.

This construct does not claim cross-language equivalence. A target may execute it
only through a compatible declared native realization; otherwise placement or
lowering must reject it. This lets Seme preserve understood surrounding meaning
without silently discarding or approximating an ecosystem/runtime operation.

| ID | Name |
| --- | --- |
| `a06d` | `NativeInvocation` |
| `a06d0` | `native_invocation.language` |
| `a06d1` | `native_invocation.callable` |
| `a06d2` | `native_invocation.signature` |
| `a06d3` | `native_invocation.arguments` |
| `a06d4` | `native_invocation.result_type` |
