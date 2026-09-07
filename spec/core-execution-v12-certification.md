# Core v12 pure-function certification

Core v12 target lowering accepts an opaque certificate, not an unchecked
canonical graph. Certification is the semantic boundary between untrusted
wire input and executable Wasm production.

The bounded certificate proves:

- exactly one `ExecutableProgram` and one declared entry `Function`;
- the entry is a member of the program's ordered function list;
- parameter references are unique and their explicit indices are contiguous
  and agree with function parameter order;
- parameter and result types are supported canonical signed modular i64 or
  Boolean types;
- the function body is exactly one `Block` containing exactly one `Return`;
- the return contains exactly one expression whose inferred type agrees with
  the declared function result;
- every reachable expression belongs to the bounded pure vocabulary;
- references and value shapes required by the executable plan are valid; and
- expression traversal is acyclic and bounded to 4096 nodes.

Calls, effects, state, mutation, locals, loops, records, and other statements
cannot occur in a certified v12 pure expression. Their presence elsewhere in a
composed envelope does not affect purity; only the selected program and its
reachable entry semantics are certified.

The certificate contains a defensive copy of the derived executable plan and
ABI. Its representation is private to the target implementation. Mutating the
caller-owned graph after certification cannot change emitted bytes, and a zero
or manually constructed certificate cannot be lowered.

This is a bounded target certificate, not a universal Core type system. Later
Core revisions may extend program membership, statement control flow, calls,
effects, and value types by defining new certificate profiles without weakening
this one.

Run `./scripts/check-core-v12-certificate.sh` for the focused rejection and
certificate-boundary gate. The complete runtime proof remains
`./scripts/check-execution-v12.sh`.
