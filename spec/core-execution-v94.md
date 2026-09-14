# Core Execution v94

Core Execution v94 preserves assignments to package-owned state as explicit
native binding mutations.

The statement records its realization language, package-scope binding, and the
canonical assigned expression. This exposes surrounding behavior without
pretending process-global mutable state is portable or pure. Go projection
reconstructs ordinary assignment syntax.

v94 adds `NativeBindingAssignment`; all prior schema identities and module
artifacts remain unchanged.
