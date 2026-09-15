# Core Execution Semantics v111

Core Execution v111 permits a general canonical ClosureConstruct to serve as an
immediately invoked typed call target. The closure and its captures remain
visible; invocation uses the existing native indirect-call mechanic whenever
the complete function signature is Go-specific.

No neutral schema changes. Run `./scripts/check-execution-v111.sh` to reproduce
the module and fixture evidence.
