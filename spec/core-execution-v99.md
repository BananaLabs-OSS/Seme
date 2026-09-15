# Core Execution v99

Core Execution v99 preserves a Go goroutine launch as an explicit native
concurrent-start statement.

The statement retains the realization language and typed invocation. It does
not claim that every language shares Go's scheduler or goroutine mechanics. Go
projection reconstructs ordinary `go call(...)` syntax and native Go remains
the authoritative realization.

v99 adds `NativeConcurrentStart`; every prior schema identity remains stable.
