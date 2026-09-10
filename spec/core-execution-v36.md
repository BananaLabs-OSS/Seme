# Core Execution v36

Core Execution v36 adds one neutral expression schema:

```text
BooleanNot(value: Boolean) -> Boolean
```

The operand is evaluated exactly once. The result is `false` for `true` and
`true` for `false`. No truthiness conversion, source operator spelling,
integer representation, exception behavior, or second-operand evaluation is
part of this schema.

Providers can use `BooleanNot` to compose comparisons already expressible by
the Core. For signed integers:

```text
left < right  = not(right <= left)
left != right = not(left <= right and right <= left)
```

This definition inherits signed i64 comparison and BooleanAnd's ordered
short-circuit semantics. It does not add a Go-specific comparison operation or
claim equality for arbitrary canonical value types.
