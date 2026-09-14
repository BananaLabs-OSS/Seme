# Core Execution v38: evaluated expression statement

Core Execution v38 adds the neutral `Evaluate` statement. It evaluates one
semantic value exactly once in normal program order and intentionally discards
the resulting value.

`Evaluate` does not make an operation pure, suppress its effects or errors,
change its result type, or imply a source-language statement-expression rule.
Providers may use it only when the native language permits the expression in
statement position and discarding the result preserves the claimed behavior.

## Frozen identities

| Identity suffix | Declaration |
|---|---|
| `a06c` | `Evaluate` |
| `a06c0` | `evaluate.value` |

Version 38 is strictly additive over v37.
