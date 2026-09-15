# Core Execution Semantics v103

Core Execution v103 preserves a standalone Go channel receive as an explicit,
typed, language-qualified operation instead of treating it as an unknown effect.

```go
<-done
```

The canonical node retains the channel expression and records Go as the owner
of blocking and scheduling mechanics. The Go projector reconstructs the native
receive statement; other language views can expose the boundary honestly.

v103 adds `NativeChannelReceive`; every prior schema identity remains stable.

Run `./scripts/check-execution-v103.sh` to reproduce the module, compiled
artifact, fixture lift, and native Go projection proof.
