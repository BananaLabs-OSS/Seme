# JavaScript provider v1

JavaScript Provider v1 is the first direct non-Go source lift into canonical
Seme execution semantics. It parses native ECMAScript modules with the pinned
Acorn parser. It does not translate JavaScript through Go; both providers emit
the same language-neutral identities, types, blocks, returns, and expressions.

## Bounded exact profile

The initial proof accepts one named synchronous exported function with:

- JSDoc `string`, `boolean`, or `bigint` parameter and result declarations;
- identifier parameters;
- total-return `if` control flow;
- string literals, parameter reads, exact string concatenation, and strict
  string equality;
- Boolean literals and short-circuit `&&` and `||` where their operands are
  within the supported Boolean profile.

The JSDoc boundary is required because JavaScript's `+` and other operators
cannot be classified exactly from untyped syntax alone. Missing annotations,
coercive `==`, async functions, generators, dynamic values, and unsupported
syntax reject with stable located diagnostics. `bigint` is reserved in the
boundary vocabulary but executable BigInt expressions are not yet claimed.
The supported string refinement contains only Unicode scalar sequences; lone
UTF-16 surrogates reject before canonical UTF-8 encoding.
JavaScript Number, coercion, truthiness, `null`, `undefined`, objects,
prototypes, exceptions, promises, and UTF-16 code-unit observation remain
unsupported rather than being silently mapped to different semantics.

## Equivalence evidence

The acceptance gate lifts idiomatic Go and idiomatic JavaScript implementations
of the same typed text function using the same semantic package identity and
revision. Their G1 source headers remain provider-specific, while the compiled
canonical SemOne archives must be byte-identical. The gate also requires:

1. deterministic repeated JavaScript lifting;
2. native Node execution producing `雪λ🦀`;
3. validation by the Kernel and Foundation;
4. lowering of the JavaScript-derived canonical graph to ABI v2 Wasm; and
5. exact matching execution of that Wasm result;
6. projection of the Go-derived canonical graph into idiomatic JavaScript;
7. native execution of that projected JavaScript; and
8. re-lifting the projection to the byte-identical canonical archive.

This proves one bounded cross-language semantic bridge, not general JavaScript
support. Canonical-to-JavaScript projection and round-trip editing are proven
only for the semantic subset listed above; broader constructs and
JavaScript-specific mechanics are subsequent provider milestones.

Run `./scripts/check-javascript-provider-v1.sh`.
