# Core Execution v97

Core Execution v97 preserves Go's two-result map lookup inside an `if`
initializer.

The lookup is evaluated exactly once as a typed native product containing the
map element and presence boolean. Both initializer bindings enter the lexical
branch scope as ordinary canonical locals. Go projection reconstructs equivalent
lookup, binding, and conditional behavior without hiding the function body.

v97 adds no schema identity; all v96 declarations and identities remain
unchanged.
