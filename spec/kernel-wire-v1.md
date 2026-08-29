# Kernel wire v1

This document defines the canonical byte representation for Seme
Kernel v1. It is deliberately a semantic graph encoding, not source text, an
AST for one presentation, or an execution target.

This encoding is frozen by the evidence and hashes in
[`KERNEL-V1-FREEZE.md`](../compiler/KERNEL-V1-FREEZE.md).

## Design boundary

The wire kernel knows only identities, schema references, revisions, fields,
values, and modules. Memory models, objects, exceptions, ownership,
concurrency, SQL, ECS, UI, machine instructions, and presentation grammars are
module-defined semantics.

Canonical files are immutable revisions. Editing creates a new revision while
ordinary edits preserve entity identities. A projection may be discarded and
reconstructed; generated source is never canonical state.

## Primitive encoding

- Byte order is independent because multi-byte integers are not used.
- Unsigned integers use shortest-form ULEB128.
- Signed integers use ZigZag followed by shortest-form ULEB128.
- Counts and byte lengths use shortest-form ULEB128.
- An identity is exactly 16 opaque bytes.
- Byte strings are length followed by uninterpreted bytes.
- There is no kernel text type. A semantic module may constrain bytes to a
  particular text encoding without making Unicode policy part of the kernel.

Decoders reject non-shortest integers, truncated values, duplicate identities,
duplicate field identities, unsorted entities or fields, unknown value tags,
invalid references, trailing bytes, and implementation resource exhaustion.

## File envelope

```text
8 bytes   magic: 53 45 4d 45 4b 31 0d 0a   (SEMEK1 CR LF)
uleb      wire version: 1
id        module identity
id        revision identity
uleb      parent count
id[]      parent revision identities, strictly lexicographically sorted
uleb      entity count
entity[]  entities, lexicographically sorted by identity
```

A content digest may index an envelope, but it is not an entity or revision
identity.

## Entity

```text
id        entity identity
id        schema identity
uleb      schema version
uleb      field count
field[]   fields, lexicographically sorted by field identity
```

Schema and field identities are stable semantic identities. Renaming or moving
an entity does not change its identity. A deleted identity is not reused.

## Values

Each value starts with one byte tag:

```text
00  unit
01  false
02  true
03  unsigned integer: uleb
04  signed integer: zigzag uleb
05  bytes: length + bytes
06  reference: identity
07  list: count + values
08  record: field count + sorted fields
09  hole: identity of the unresolved semantic decision
```

Lists are ordered. Records use the same stable field identities and ordering
rule as entities. Maps, sets, strings, tuples, unions, numbers with widths, and
all domain values are schemas built from these values rather than extra kernel
tags.

## References and preservation

Every wire reference resolves to an entity in this envelope. Cross-module
targets are represented by a local import entity whose versioned schema names
the external module, revision requirements, and target identity. Resolution of
that import is semantic-module validation above the wire decoder.

A conforming read-modify-write implementation must preserve unknown entities,
unknown fields, unknown schema versions, and unknown value structure exactly at
the semantic-value level. It may canonicalize their byte encoding. It must not
claim to validate semantics it does not understand.

## Bootstrap structural schema

The wire decoder recognizes no domain schema. A small bootstrap module proves
that Schema, Field, and Module declarations can describe themselves without
making their future semantic vocabulary part of the wire kernel.

The larger semantic foundation in `kernel-meta-v1.md` is ordinary versioned
module data. Generic schema, import, constraint, effect, provenance, and
refinement validation may evolve above the frozen decoder.

## Determinism and diagnostics

Validation order is envelope order, then entity identity, then field identity,
then list position. A diagnostic contains a stable rule identity, revision
identity, entity identity, a field/list path, and structured value arguments.
Human wording belongs to a diagnostic projection and is not used to compare
compiler generations.

The canonical report shape and bootstrap delivery contract are specified in
[`diagnostics-v1.md`](diagnostics-v1.md).

## Target contract

Reading this format does not imply that a target can execute every represented
mechanic. A target reports each required construct as `exact`, `refined`,
`native`, `emulated`, `guarded`, `adapted`, `embedded`, or `impossible` before
lowering performs effects.
