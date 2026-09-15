# Core Execution Semantics v107

Core Execution v107 adds a typed native indirect-invocation operation. It
records the owning language, callable expression, native signature, ordered
arguments, and result type. This lets canonical behavior invoke function-valued
parameters, locals, fields, and method values without pretending that every
language shares Go's calling mechanics.

Go projection restores ordinary indirect-call syntax. Other language views can
surface the explicit native boundary. Run `./scripts/check-execution-v107.sh`
to reproduce the module, compiled artifact, fixture lift, and Go execution.
