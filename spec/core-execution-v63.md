# Core Execution v63: typed native field assignment

Core Execution v63 adds `NativeFieldAssignment`, an explicit language-owned
mutation boundary with a language, field name, receiver expression, and value
expression.

The Go provider uses it for assignments such as `state.Name = name`. Go retains
its pointer, addressability, aliasing, and mutation mechanics. Go projection
restores ordinary field-assignment syntax. JavaScript and Lua projections expose
the same operation through explicit Seme runtime adapters and do not claim that
their object models are automatically equivalent to Go's.

This keeps neutral Seme free of a preferred language's memory model while
allowing surrounding control flow and values to remain canonical.
