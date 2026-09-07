# Core Execution Semantics v8

Core Execution v8 additively extends v7 with the first provider-neutral,
compositional function-body structure.

| Identity | Schema | Meaning |
|---|---|---|
| `...9080` | `Block` | An ordered list of statement references |
| `...9081` | `Return` | An ordered list of returned expression references |

`Function.body` may reference a `Block` in v8. A block preserves semantic
execution order; it does not preserve source punctuation, formatting, or a
particular language's statement syntax. A `Return` terminates its containing
function activation and evaluates its values from left to right.

Return values are expressions from Core Execution's compositional vocabulary.
This separates statement structure from expression structure: an integer
addition, parameter read, literal, field read, record construction, conditional
value, or function call is not itself a statement merely because one source
language permits expression statements.

The empty return-value list represents a return from a function whose result is
unit. Multiple values are ordered. Validation and execution profiles must check
the returned values against the function result contract; v8 does not silently
discard or synthesize values.

V8 does not yet define local bindings, assignment, expression statements,
loops, or statement-level conditionals. Providers must reject those constructs
for exact execution rather than encode them as application-specific shapes.

All v1-v7 schema declarations and checked artifacts remain unchanged. Existing
functions whose bodies directly reference expressions remain valid under their
original revision contracts; v8 consumers may require structured bodies when
claiming v8 fidelity.
