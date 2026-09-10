# Core Execution Semantics v36

Version 36 adds `BooleanNot`, a language-neutral logical negation over one
boolean expression. It does not encode any source-language operator spelling,
truthiness conversion, integer comparison, or evaluation of a second operand.

Providers may compose existing exact semantics through it. In particular,
signed integer `left < right` is `not(right <= left)`, while signed integer
`left != right` is `not(left <= right and right <= left)`.
