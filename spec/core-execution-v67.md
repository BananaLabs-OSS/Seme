# Core Execution v67: typed native addresses

Core Execution v67 adds `NativeAddress`, recording the owning language, the
canonical operand, and the language-owned pointer result type. The Go provider
uses it for unary `&` after type checking, and Go projection restores a native
address expression.

JavaScript and Lua projections retain the operation through explicit
`Seme.nativeAddress` and `Seme.native_address` adapters. They do not pretend
that JavaScript or Lua has Go pointer mechanics. The operand remains ordinary
canonical meaning whenever it can be represented; only address ownership and
the pointer type remain in the Go realization boundary.
