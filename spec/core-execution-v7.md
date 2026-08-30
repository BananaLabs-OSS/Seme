# Core Execution Semantics v7

Core Execution v7 additively extends v6 with `IntegerLiteral` (`...9070`).
Its `integer_literal.bits` field stores the width-bounded two's-complement bit
pattern as an unsigned value, while `integer_literal.type` identifies the
`IntegerType` that determines width, signedness, and overflow semantics.

This keeps literal storage independent of presentation spelling. Go decimal,
hexadecimal, octal, binary, and underscore-separated integer tokens resolve to
the same canonical bits when they denote the same value.
