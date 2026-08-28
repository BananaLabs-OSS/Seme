# Kernel wire v1 candidate

This document defines the candidate canonical byte representation for Seme
Kernel v1. It is deliberately a semantic graph encoding, not source text, an
AST for one presentation, or an execution target.

The candidate remains unfrozen until every gate in `kernel-v1.md` passes.

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

Every reference resolves either to an entity in this envelope or to an import
declared by the module entity. Reference validity is checked before semantic
module validation.

A conforming read-modify-write implementation must preserve unknown entities,
unknown fields, unknown schema versions, and unknown value structure exactly at
the semantic-value level. It may canonicalize their byte encoding. It must not
claim to validate semantics it does not understand.

## Bootstrap meta-schema

The wire decoder recognizes no domain schema. Kernel validation begins from a
small versioned meta-schema module whose checked canonical envelope defines:

- schema declarations;
- field declarations and cardinality;
- module imports and exports;
- constraints and their stable diagnostic rule identities;
- effects and required capabilities;
- provenance and refinement relationships.

The meta-schema is data encoded by this wire format. Its identity constants and
initial canonical bytes will be frozen together only after the self-description
and external-module conformance fixtures pass.

## Determinism and diagnostics

Validation order is envelope order, then entity identity, then field identity,
then list position. A diagnostic contains a stable rule identity, revision
identity, entity identity, a field/list path, and structured value arguments.
Human wording belongs to a diagnostic projection and is not used to compare
compiler generations.

## Target contract

Reading this format does not imply that a target can execute every represented
mechanic. A target reports each required construct as `exact`, `refined`,
`native`, `emulated`, `guarded`, `adapted`, `embedded`, or `impossible` before
lowering performs effects.
