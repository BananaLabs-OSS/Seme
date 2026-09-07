# Core Execution Semantics v11

Core Execution v11 additively extends v10 with boolean literals and
short-circuit boolean conjunction.

| Identity | Schema | Fields |
|---|---|---|
| `...90b0` | `BooleanLiteral` | `value (...9b00)` |
| `...90b1` | `BooleanAnd` | `left (...9b10)`, `right (...9b11)` |

`BooleanLiteral.value` is a canonical Kernel boolean, never an integer truthy
value. `BooleanAnd` evaluates its left operand first. When that value is false,
the result is false and the right operand is not evaluated. When it is true,
the right operand is evaluated and becomes the result. Both operands must have
canonical Boolean type.

Short-circuiting is observable whenever the right expression may reject,
diverge, or perform an effect in a later execution revision. A target may not
replace `BooleanAnd` with an eager operator unless it proves the skipped
evaluation is unobservable under the declared effect contract.

Go's typed `true`, `false`, and `&&` constructs map exactly to these semantics
inside the bounded profile. Other ecosystems may use different surface syntax;
the canonical behavior is independent of that presentation.

All v1-v10 declarations and checked artifacts remain unchanged. Boolean OR,
negation, and other unintroduced operations remain unsupported.
