# Core Execution Semantics v106

Core Execution v106 retains Go variadic method invocation as the existing typed
native method-call operation. The callable identity explicitly records spread
application, while argument evaluation and the surrounding behavior remain
canonical. Go owns variadic slice expansion.

No neutral schema changes. Run `./scripts/check-execution-v106.sh` to reproduce
the module, compiled artifact, fixture lift, and native Go projection proof.
