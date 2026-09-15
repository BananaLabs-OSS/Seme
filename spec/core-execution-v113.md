# Core Execution Semantics v113

Core Execution v113 preserves Go `select` receive-assignment bindings as typed,
positioned local identities scoped to their communication clause. Both the
received value and the optional channel-open boolean can be used by canonical
clause bodies and projected back to an exact Go receive assignment.

This remains an explicitly Go-native concurrency construct: Seme preserves its
typed meaning and native realization boundary without claiming that every
language implements Go channel selection. Run `./scripts/check-execution-v113.sh`
to reproduce the module and fixture evidence.
