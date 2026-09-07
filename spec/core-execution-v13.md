# Core Execution Semantics v13

Core Execution v13 extends the compositional statement and expression model.
It is not tied to a consumer, function name, or recognized whole-function
shape.

## Additive vocabulary

- `If(condition, then, else)` is a statement with two required `Block`
  references. Requiring both branches makes executable totality explicit.
- `BooleanOr(left, right)` evaluates left first and evaluates right only when
  left is false.
- `StringEqual(left, right)` compares exact UTF-8 byte sequences.
- `StringConcat(left, right)` constructs the ordered concatenation of two text
  expressions.

`StringLiteral` and `StringType` retain their earlier canonical identities.
No v1-v12 declaration or field changes.

## Bounded Go lift

The provider recursively analyzes typed expressions and total-return blocks.
An idiomatic Go form such as `if condition { return a }; return b` becomes one
canonical `If` whose then and else blocks each terminate in `Return`. Nested
forms and explicit `else` blocks are handled the same way. Initializers,
non-returning paths, locals, loops, and unreachable statements reject rather
than acquiring guessed meaning.

## Exact runtime profile

The v12 pure-function ABI continues to expose canonical Boolean and signed
modular-i64 parameters and results. Its certificate now validates and lowers
nested `Block -> If -> Block` trees recursively. Wasm structured `if` results
preserve branch result types, and Boolean OR preserves source short-circuit
ordering.

V13 text execution is deliberately bounded to closed UTF-8 literals,
concatenation, and equality. The target validates UTF-8, enforces a 4096-byte
constructed-value bound, and resolves these pure closed values while lowering.
String ABI parameters/results, mutable strings, indexing, and locale-aware
operations remain unsupported. This is exact support for the declared subset,
not a claim of general Go string support.

Run `./scripts/check-execution-v13.sh` for native differential tests,
canonical validation, deterministic lifting/lowering, standalone Wasm and Pulp
execution, predecessor regeneration, and contamination checks.
