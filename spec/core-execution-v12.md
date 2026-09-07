# Core Execution Semantics v12

Core Execution v12 freezes the first generic structured pure-function runtime
profile over the unchanged additive v11 vocabulary. It adds no schema: the
meaning was already expressible as `Program -> Function -> Block -> Return`
with compositional expressions. V12 supplies validation, ABI derivation, and
execution evidence without putting a target wire format into Core semantics.

## Bounded pure profile

- At most 32 ordered parameters, each canonical signed modular i64 or Boolean.
- Exactly one result, canonical signed modular i64 or Boolean.
- Exactly one `Return` in the entry function's body `Block`.
- Pure expressions supported through v11: parameter reads, integer/boolean
  literals, addition, multiplication, ordered subtraction, signed less-equal,
  and short-circuit BooleanAnd.
- No effects, mutable state, locals, calls, loops, records, or variable-width
  values in this profile.

## Derived Pulp ABI

Request fields occur in canonical parameter-index order. An i64 occupies eight
little-endian bytes and a Boolean occupies one byte (`00` false, `01` true).
The request length must equal the exact derived size. The response is the single
result in the same scalar encoding. A noncanonical Boolean request byte rejects.

The ABI is target evidence, not canonical program meaning. Another target can
realize the same function through a native calling convention while preserving
the semantic signature.

The emitted Wasm exports memory and the standard Pulp allocator, lifecycle, and
provider-call functions. It runs directly in a WebAssembly runtime and through
Pulp's normal manifest, cell, allocation, and provider dispatch path.
