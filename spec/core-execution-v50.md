# Core Execution v50: assignment initializers and nested return flow

Core Execution v50 preserves two common Go control-flow forms:

- a multi-result call in an `if` initializer may assign existing mutable
  bindings; the call is evaluated once and product items update their original
  places in order;
- a conditional branch containing a nested early return receives the remaining
  continuation on every path that falls through.

This makes nested error-fallback code compositional without duplicating calls
or treating a syntactically nested return as an unconditional return. Go type
assertions remain a distinct unsupported mechanic. No schema is added beyond
v49.
