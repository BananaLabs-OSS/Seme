# Core Execution Semantics v101

Core Execution v101 preserves Go `select` control flow as typed,
language-qualified structure. Each case explicitly records whether it receives,
sends, or is the default, plus its channel, optional sent value, and canonical
body.

Go remains the owner of scheduling and channel-selection mechanics. Seme can
nevertheless inspect case structure and project it back to valid Go without
flattening the whole function into a native island. Receive assignments remain
an explicit future extension; v101 covers receive-only, send, and default cases.

v101 adds `NativeSelectCase` and `NativeSelect`; every prior schema identity
remains stable.

Run `./scripts/check-execution-v101.sh` to reproduce the module, compiled
artifact, fixture lift, and native projection proof.
