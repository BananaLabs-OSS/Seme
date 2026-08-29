# Structured diagnostics module v1 candidate

Diagnostics are semantic data. Human-readable messages, colors, source
snippets, and localization are projections and are not canonical.

The initial Kernel meta-schema identities are:

| Identity | Entity |
|---|---|
| `00000000000000000000000000000019` | DiagnosticRule |
| `0000000000000000000000000000001a` | Diagnostic |
| `0000000000000000000000000000001b` | DiagnosticReport |

## DiagnosticRule fields

| Identity | Meaning | Shape |
|---|---|---|
| `00000000000000000000000000000190` | stable symbolic name | bytes |
| `00000000000000000000000000000191` | argument shapes | ordered list(record ValueShape) |

A rule identity defines the failure. Its name assists projections but is not
used to compare diagnostics.

The bootstrap validator uses the stable parent rule `kernel.invalid_input` with
a structured unsigned kind argument: `0` means a decoded envelope violated
structural relationships and `1` means the input could not be decoded
canonically. Later validators may refine these failures while retaining the
parent rule for consumers that only need the broad classification.

## Diagnostic fields

| Identity | Meaning | Shape |
|---|---|---|
| `000000000000000000000000000001a0` | rule | reference DiagnosticRule |
| `000000000000000000000000000001a1` | revision identity | bytes, exactly 16 |
| `000000000000000000000000000001a2` | entity identity | bytes, zero or exactly 16 |
| `000000000000000000000000000001a3` | semantic path | ordered list(record PathSegment) |
| `000000000000000000000000000001a4` | arguments | ordered list(value) |
| `000000000000000000000000000001a5` | severity | unsigned |

Entity identities are bytes rather than references because a diagnostic must
be able to locate a missing, duplicate, or otherwise invalid entity. Empty
bytes mean that validation failed before an entity location existed.

An all-zero revision identity means the input failed before its claimed
revision could be trusted. It is a location sentinel, never an allocated
revision identity.

Severity values are `0` error, `1` warning, and `2` information. Severity does
not affect diagnostic identity or ordering.

## PathSegment record

| Identity | Meaning | Shape |
|---|---|---|
| `00000000000000000000000000002100` | kind | unsigned |
| `00000000000000000000000000002101` | identity | optional bytes, exactly 16 |
| `00000000000000000000000000002102` | index | optional unsigned |

Kinds are `0` entity, `1` field, `2` list element, and `3` nested record field.
The required location is semantic and remains stable when a textual projection
is reformatted. Projection-specific source ranges may be derived separately.
An all-zero identity in a path segment means the validator reached that
semantic level but the malformed input did not provide a trustworthy identity.

## DiagnosticReport fields

| Identity | Meaning | Shape |
|---|---|---|
| `000000000000000000000000000001b0` | status | unsigned |
| `000000000000000000000000000001b1` | diagnostics | ordered list(reference Diagnostic) |

Reports use the ordinary canonical Kernel envelope. Diagnostics are ordered by
validation phase, entity identity, field identity, list position, then rule
identity. Implementations stop at a specified phase boundary; they must not
reorder failures based on human wording.

## Bootstrap invocation

During the frozen K0 transition, validators retain their existing process
statuses: `0` success, `64` invalid invocation, `65` semantic/malformed input,
and `74` I/O failure. With only an input argument, validation produces no file.
With the existing optional result-path argument, the validator writes the
canonical input on success or a canonical DiagnosticReport on semantic failure.
The process status remains a shell-compatible summary.

If the semantic result cannot be written, the validator returns `74`. Selecting
the result mode may not otherwise affect validation behavior.
