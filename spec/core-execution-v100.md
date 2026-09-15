# Core Execution Semantics v100

Core Execution v100 preserves a Go channel send as an explicit, typed,
language-qualified operation instead of rejecting the containing function as a
native island.

```go
done <- struct{}{}
```

The canonical `NativeChannelSend` node records the source language, channel
expression, and value expression. This keeps channel mechanics owned by the Go
realization while making the operation structurally visible to Seme. The Go
projector reconstructs valid native Go, while other projectors can expose an
honest native boundary rather than inventing equivalent semantics.

v100 adds `NativeChannelSend`; every prior schema identity remains stable.

Run `./scripts/check-execution-v100.sh` to reproduce the module, compiled
artifact, fixture lift, native projection test, and prior-version compatibility
checks.
