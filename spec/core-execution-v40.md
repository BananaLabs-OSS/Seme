# Core Execution v40: typed native method invocation

Core Execution v40 adds `NativeMethodInvocation`. It preserves a native value
receiver separately from ordered explicit arguments, together with an opaque
runtime language, collision-resistant callable identity, exact native
signature, and canonical result type.

The receiver is evaluated exactly once before the arguments. This node does not
claim portability: a target must resolve the complete native identity through a
compatible adapter or reject placement. The initial Go provider admits only
non-variadic, non-interface, value-receiver methods with one supported scalar
result. Pointer receivers remain native islands because their mutation cannot
yet be reflected into a canonical place honestly.

| ID | Name |
| --- | --- |
| `a06e` | `NativeMethodInvocation` |
| `a06e0` | `native_method_invocation.language` |
| `a06e1` | `native_method_invocation.callable` |
| `a06e2` | `native_method_invocation.signature` |
| `a06e3` | `native_method_invocation.receiver` |
| `a06e4` | `native_method_invocation.arguments` |
| `a06e5` | `native_method_invocation.result_type` |
