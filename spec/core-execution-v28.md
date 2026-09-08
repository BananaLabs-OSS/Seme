# Core Execution v28: immutable lexical closures

Core v28 models closures as typed callable values with explicit immutable
environments. It adds function types, capture bindings and reads, closure
construction, and indirect invocation. Capture discovery, escape analysis, and
native syntax remain adapter responsibilities.

The bounded proof uses `MakeAdder(base)` to return a unary function, `Apply` to
invoke a passed function indirectly, and `Run(base, value)` to compose both.
Positive and negative captured integers establish that the environment survives
its declaring call and is not replaced by a compile-time constant.
