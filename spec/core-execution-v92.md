# Core Execution v92

Core Execution v92 preserves explicit Go type conversions as typed native
mechanics.

The source and target remain checked by Go's type system. Seme retains the
canonical operand, target spelling, result identity, and native realization
requirement. Go projection reconstructs ordinary conversion syntax. This
supports conversions such as `string(bytes)` and `uint32(integer)` without
claiming that their Go-specific mechanics are universally identical.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
