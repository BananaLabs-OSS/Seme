# Core Execution v54: native products and contextual nil

Core Execution v54 completes typed application-boundary symmetry for function
products and returns:

- each unsupported application-owned item in a multi-result signature retains
  its exact Go-native type identity;
- `nil` in an explicit return position is interpreted using that result's
  compile-time Go type;
- the contextual zero becomes either an existing neutral zero or an explicit
  typed Go-native default value.

This preserves Go's typed nil mechanics without adding them to neutral Core or
guessing from syntax alone. No schema is added beyond v53.
