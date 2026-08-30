# Core Execution v7 freeze audit

| Property | Result | Evidence |
|---|---|---|
| Additive evolution | pass | Every v6 schema and field identity remains unchanged. |
| Literal semantics | pass | `IntegerLiteral` records width-bounded bits and an explicit `IntegerType`. |
| Source analysis | pass | Go integer tokens are parsed as signed 64-bit constants before canonical emission. |
| Recursive emission | pass | A literal nested within two additions receives a deterministic semantic-path identity. |
| Backend validation | pass | Wasm lowering validates the literal type and emits signed `i64.const` from canonical bits. |
| Runtime continuity | pass | Go, Node Wasm, and Pulp preserve the quota behavior with the nested `+ 0` expression. |

This freeze covers signed 64-bit Go integer tokens representable by
`strconv.ParseInt`. Constant folding, arbitrary-precision untyped constants,
conversions, unary expressions, and additional integer widths remain future
semantic increments.
