# Patch Module v1

Patch Module v1 defines transactional semantic editing above frozen Kernel v1
and Semantic Foundation v1.

## Stable identities

| Identity | Meaning |
|---|---|
| `00000000000000000000000000005000` | Patch module |
| `00000000000000000000000000005010` | Patch schema |
| `00000000000000000000000000005011` | RenameDeclaration schema |
| `00000000000000000000000000005100` | patch.author |
| `00000000000000000000000000005101` | patch.base_revision |
| `00000000000000000000000000005102` | patch.operations |
| `00000000000000000000000000005110` | rename.target |
| `00000000000000000000000000005111` | rename.field |
| `00000000000000000000000000005112` | rename.expected_value |
| `00000000000000000000000000005113` | rename.replacement |
| `00000000000000000000000000005601` | patch.stale_revision rule |
| `00000000000000000000000000005602` | patch.unknown_target rule |
| `00000000000000000000000000005603` | patch.unknown_field rule |
| `00000000000000000000000000005604` | patch.precondition_failed rule |
| `00000000000000000000000000005605` | patch.duplicate_write rule |
| `00000000000000000000000000005606` | patch.invalid_candidate rule |
| `00000000000000000000000000005700` | Patch rejection Diagnostic role |
| `00000000000000000000000000005701` | Patch rejection DiagnosticReport role |

Patch identity is the ordinary Kernel entity identity of a Patch instance.

## Patch

A patch contains stable patch identity, author/provenance identity, base
revision identity, and an ordered list of operations. Patch identity is not a
content digest and is never reused.

V1 defines one operation:

```text
RenameDeclaration
  target          bytes containing the declaration entity identity
  field           identity of its name field
  expected_value  bytes precondition
  replacement     bytes
```

Declaration names are non-empty byte sequences in v1. A rename whose
replacement is empty reaches candidate validation and rejects with
`patch.invalid_candidate`.

The operation is intentionally field-addressed. The Patch module does not
assume that all languages use one universal name field.

## Transaction

Application follows this exact order:

1. require the patch base revision to equal the workspace revision;
2. validate the Patch entity through Semantic Foundation v1;
3. resolve every target and field identity;
4. evaluate all preconditions against the unchanged base workspace;
5. apply every operation to an isolated candidate;
6. validate the complete candidate;
7. derive and commit one new immutable revision;
8. otherwise return one deterministic diagnostic and leave the workspace
   byte-for-byte unchanged.

Operations never observe partial results from earlier operations in the same
patch when evaluating preconditions. Two operations attempting to write the
same entity field conflict and reject.

## Required diagnostics

- `patch.stale_revision`
- `patch.unknown_target`
- `patch.unknown_field`
- `patch.precondition_failed`
- `patch.duplicate_write`
- `patch.invalid_candidate`

Diagnostics carry the patch revision, target entity where available, operation
list index, field identity where available, and structured expected/actual
values. The canonical report is an ordinary Kernel envelope whose module role
is `...5701`, whose revision is the rejected workspace revision, and whose
status is `3` (rejected). Expected and actual are always present as two bytes
arguments; unavailable values use empty bytes. The Patch applicator returns
process status `65` after publishing this report and never publishes its
isolated candidate on rejection.

## Revision derivation

The new revision identity is the first 128 bits of SHA-256 over this exact byte
transcript:

```text
"seme.patch.revision.v1\0"
|| base revision identity (16 bytes)
|| isolated canonical candidate envelope with a zero revision placeholder
```

Patch v1 workspaces retain the Patch and ordered operation entities unchanged
inside the isolated candidate, so the final transcript includes both the
canonical patch value and complete canonical candidate value without a second
serialization. The digest identifies the immutable revision; stable entity
identities remain unchanged.
