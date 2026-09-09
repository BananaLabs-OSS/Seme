# JavaScript UAB-03 evidence

This candidate cell composes lexical locals, mutable assignment, modular i64
addition, signed comparison, short-circuit Boolean `&&` and `||`, a conditional
inside a bounded `while`, and a conditional function-level early return.

JavaScript `bigint` is an adapted realization of canonical i64. Arithmetic is
projected as `BigInt.asIntN(64, ...)` so canonical two's-complement wrapping is
preserved natively. Inputs pass exact bigint/range and Boolean checks; Number,
coercion, and general JavaScript truthiness are not admitted as Core meaning.
Both logical operators have Boolean operands, so JavaScript operand-returning
and truthiness behavior is deliberately outside this bounded mapping.

`scripts/check-javascript-uab-03.sh` proves the same valid observations through
the original and projected Node source, the independent structural canonical
evaluator, standalone Wasm, and pinned Pulp over six boundary/adversarial cases
plus 100 reproducibly generated cases. Invalid checked indexes behind false
`&&` and true `||` prove that the right operand is skipped, while matching
non-skipped cases reject through all five realizations. The gate also checks
canonical malformed types, malformed target encodings, byte-identical
projection/re-lift, and located rejection of unwrapped BigInt arithmetic.

The loop body contains a conditional mutation. Returning from inside a loop is
not part of this bounded cell; the later UAB-11 cumulative application still
requires its separately specified loop-internal early rejection.

Independent integration review awarded all five evidence classes after the
complete gate and UAB-01/UAB-02 regressions passed.
