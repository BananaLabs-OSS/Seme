# Core Execution v57: typed native comparisons

Core Execution v57 preserves Go comparisons whose operand mechanics are not
neutral Core mechanics.

Go type checking determines each operand's contextual type. Seme retains any
required assignment conversion, typed `nil`, operator, operand order, and exact
Go type identities in an explicit native comparison operation. Existing neutral
string and stable i64 comparisons remain canonical. This supports pointers,
interfaces, `int`, named values, and other Go-owned comparable or ordered types
without conflating their runtime rules with Core. No schema is added beyond
v56.
