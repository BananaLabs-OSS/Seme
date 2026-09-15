# Core Execution Semantics v110

Core Execution v110 generalizes immutable Go closures into canonical closure
construction: typed parameters and captures, multi-statement block bodies, local
bindings, portable and native Go types, and exact Go projection. Mutable and
nested captures remain separate, explicit boundaries.

No neutral schema changes are required because the existing language-neutral
FunctionType, CaptureBinding, CaptureRead, and ClosureConstruct contracts already
express this meaning. Run `./scripts/check-execution-v110.sh` to reproduce the
module and fixture evidence.
