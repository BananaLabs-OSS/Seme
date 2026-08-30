# Core Execution Semantics v5

Core Execution v5 additively extends v4 with:

| Identity | Schema |
|---|---|
| `...9050` | `StringLiteral` |
| `...9051` | `StringIsEmpty` |
| `...9052` | `Conditional` |

A conditional references its condition, then value, and else value. Both arms
must satisfy the surrounding expected type. This lets Result success/error
selection remain canonical rather than becoming target-only branching.
