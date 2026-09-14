# Core Execution v89

Core Execution v89 preserves Go calls that explicitly expand a final slice
into a native variadic function.

The invocation retains the declared Go signature, typed arguments, native
callable identity, and an explicit ellipsis realization marker. Go projection
restores `values...`; non-Go targets must provide an adapter with equivalent
mechanics or reject placement.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
