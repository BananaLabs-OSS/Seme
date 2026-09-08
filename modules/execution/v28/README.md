# Core Execution Semantics v28

Version 28 adds immutable lexical closures. Function types describe callable
signatures structurally. Closure construction records parameters, body, and an
explicit immutable environment of capture bindings. Indirect calls consume a
callable value without introducing source-language closure rules into Core.
