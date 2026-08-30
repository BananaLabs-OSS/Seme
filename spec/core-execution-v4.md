# Core Execution Semantics v4

Core Execution v4 additively extends v3 with strings, byte sequences, and
explicit fallibility.

| Identity | Schema |
|---|---|
| `...9040` | `StringType` |
| `...9041` | `BytesType` |
| `...9042` | `ResultType` |
| `...9043` | `ResultOk` |
| `...9044` | `ResultError` |

`ResultType` references its success and error types. `ResultOk` and
`ResultError` reference that result type plus the selected value. This keeps
fallibility in canonical semantics instead of encoding it as a magic integer,
nullable pointer, exception convention, or target ABI status.

String and byte values are semantic sequences. Encoding, allocation,
ownership, maximum length, and concrete `(pointer, length)` representation are
properties of a target mapping and must be reported there.
