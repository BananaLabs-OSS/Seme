# Core Execution Semantics v16

Core Execution v16 freezes the first executable closed multi-function program
profile. An `ExecutableProgram` owns an ordered, unique set of `Function`
members and identifies one member as its entry. `FunctionCall(callee,
arguments)` invokes another member with arguments evaluated left-to-right and
bound by parameter position. The call produces the callee's declared result.

Go package-level calls and JavaScript module-local calls are exact source views
when the callee has a supported statically known signature. Both providers
emit the same canonical function identities, membership, entry, parameters,
and call expressions. JavaScript projection emits ordinary module-local
functions and exports only the canonical entry; re-lifting recovers the same
canonical program.

The v16 pure Wasm realization accepts only a closed acyclic call graph of at
most 64 functions, 32 parameters per function, and bounded total expansion.
Every callee must belong to the program. Recursive calls, external calls,
malformed arity, invalid membership, and graphs exceeding the traversal budget
reject before artifact emission. Callee bodies in this initial realization
must normalize to a pure returned expression; canonical calls themselves are
not erased. The Wasm target may inline them as a physical optimization after
certification.

This milestone does not claim recursion, indirect calls, effects, imports, or
dynamic dispatch. Those require additional canonical mechanics and runtime
realizations rather than source-shaped exceptions.
