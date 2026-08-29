# Package Contract v1

Package Contract v1 represents the boundary around an independently usable
software package. It is canonical module `...b000`, revision `...b001`.

| Identity | Schema |
|---|---|
| `...b010` | Package |
| `...b011` | TypedInterface |
| `...b012` | Dependency |
| `...b013` | RuntimeAssumption |
| `...b014` | FidelityMapping |

A Package records its ecosystem name and revision plus ordered interface,
dependency, required-effect, runtime-assumption, and fidelity-mapping lists.
The lists are required even when empty: absence of dependencies or effects is
an explicit, testable claim rather than missing analysis.

TypedInterface connects an exported name to the stable canonical Function and
its parameter/result type identities. Dependency records a requested package
and resolution. Required-effect values reference Foundation Effect entities
when present. RuntimeAssumption records mechanics that observable behavior
depends on. FidelityMapping connects source and canonical entities using the
stable fidelity values from Target Contract v1, with evidence.

The generic Foundation shape for required effects is intentionally an
unconstrained reference list because a composed graph may carry effects from a
separately versioned semantic module. Package-specific conformance must verify
that every nonempty item is an Effect entity from a declared module. The v1
quota fixture proves the empty case.

## First application profile

The ordinary `example.com/seme-quota-proof` Go package exports:

```go
func Admit(current int64, delta int64, limit int64) bool {
    return current+delta <= limit
}
```

Its canonical package contract declares:

- one typed `Admit` interface tied to the provider's stable Function identity;
- zero dependencies and zero effects;
- signed 64-bit modular arithmetic as a runtime assumption;
- exact Go-to-Core-Execution fidelity with type-check and differential evidence.

The source remains untouched and passes its normal Go tests. Core Execution v2
executes the lifted policy without Go at runtime and compares normal, rejection,
negative, overflow, and zero cases with Go. Unsupported source, absent package
evidence, and malformed invocation reject explicitly.

This is the first package/application contract proof. It does not establish
dependency resolution, effect execution, records, cross-package calls, or a
general Go package loader. Those extend this contract from new application
needs rather than entering the Kernel.
