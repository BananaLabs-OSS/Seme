# Core Execution Semantics v19

Core Execution v19 adds the provider-neutral `When` statement. `When` owns a
Boolean condition and a nested block. It executes the body once when true and
falls through unchanged when false. Unlike the existing total-return `If`, it
does not invent an else branch or require either path to terminate the
surrounding function.

Ordinary one-sided Go and JavaScript `if` statements are exact source views.
Together with v18 mutable places, they express conditional state transitions
without importing either source language's syntax into canonical Seme. The
bounded proof mutates a signed-i64 place, projects it as native JavaScript
`bigint`, and preserves byte-identical Go/JavaScript canonical meaning.

The certified Wasm state realization now accepts signed-i64 places alongside
Boolean and string places. It emits an i64 local and a structured void `if`,
validates lexical visibility and types before emission, and executes both
outcomes standalone and through Pulp. The exact two's-complement modular-i64
ABI is retained inside the state-capable v2 provider envelope.

This profile does not yet claim `else` with fallthrough, `else if` mutation,
mutation through function calls, integer arithmetic over place reads, or
`break` and `continue`.
