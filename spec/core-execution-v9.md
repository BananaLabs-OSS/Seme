# Core Execution Semantics v9

Core Execution v9 additively extends v8 with typed integer multiplication.

| Identity | Schema | Fields |
|---|---|---|
| `...9090` | `IntegerMultiply` | `left (...9900)`, `right (...9901)`, `type (...9902)` |

Both operands are compositional expression references. Evaluation is ordered
left then right. The referenced `IntegerType` determines width, signedness, and
overflow mechanics; for the existing signed i64 modular profile, multiplication
retains the low 64 bits. It does not depend on a host language's unchecked
signed-overflow behavior.

Providers map a source multiplication operator only after native type
resolution proves this semantic operation. Operator spelling alone is not
authority. Targets must either realize the declared integer mechanics or report
a weaker fidelity or impossibility.

All v1-v8 declarations and checked artifacts remain unchanged. Subtraction is
not part of v9 and remains unsupported pending its own semantic revision.
