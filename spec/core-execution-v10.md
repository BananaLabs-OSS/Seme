# Core Execution Semantics v10

Core Execution v10 additively extends v9 with typed integer subtraction.

| Identity | Schema | Fields |
|---|---|---|
| `...90a0` | `IntegerSubtract` | `left (...9a00)`, `right (...9a01)`, `type (...9a02)` |

Both operands are compositional expression references and are evaluated left
then right. Subtraction preserves operand order: `left - right` is never
normalized by exchanging its children. The referenced `IntegerType` determines
width, signedness, and overflow mechanics. Signed i64 modular subtraction
retains the low 64 bits.

Providers map source subtraction only after native type resolution proves this
operation. Targets must preserve both the declared arithmetic mechanics and the
ordered operands.

All v1-v9 declarations and checked artifacts remain unchanged. Division and
other unintroduced operations remain unsupported.
