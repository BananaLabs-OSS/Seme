# Core Execution Semantics v105

Core Execution v105 represents a Go method value as an explicit typed native
operation. The canonical node records the owning language, method identity,
native signature, bound receiver expression, and resulting function type.

This preserves the essential distinction between selecting a bound method and
invoking it. Go owns receiver binding and method-set mechanics; Seme owns the
typed operation and its position in the surrounding canonical behavior. Go
projection restores native method-value syntax, while other projections can
expose the declared language boundary without pretending it is portable.

Run `./scripts/check-execution-v105.sh` to reproduce the module, compiled
artifact, fixture lift, and native Go projection proof.
