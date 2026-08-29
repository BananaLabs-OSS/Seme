# Kernel meta-schema v1 candidate

This is the candidate semantic bootstrap for Kernel wire v1. It defines the
smallest data model needed to interpret schemas and modules. The identities are
allocated and must never be silently reused, but they are not frozen until the
Kernel v1 freeze gates pass.

Identity literals below are 16-byte hexadecimal values. Their numeric pattern
is only an audit convenience; consumers must treat them as opaque.

## Schema identities

| Identity | Meaning |
|---|---|
| `00000000000000000000000000000001` | kernel module entity |
| `00000000000000000000000000000010` | Schema |
| `00000000000000000000000000000011` | Field |
| `00000000000000000000000000000012` | Module |
| `00000000000000000000000000000013` | Import |
| `00000000000000000000000000000014` | Constraint |
| `00000000000000000000000000000015` | Effect |
| `00000000000000000000000000000016` | Capability |
| `00000000000000000000000000000017` | Provenance |
| `00000000000000000000000000000018` | Refinement |
| `00000000000000000000000000000019` | DiagnosticRule |
| `0000000000000000000000000000001a` | Diagnostic |
| `0000000000000000000000000000001b` | DiagnosticReport |

`Schema` is an instance of itself. Every other row is an entity whose schema is
`Schema`. This finite self-reference is intentional: the wire decoder can read
all entities structurally, then the meta-validator can interpret Schema using
the same declarations it validates.

## Field identities

| Identity suffix | Field | Value shape |
|---:|---|---|
| `0100` | schema.name | bytes |
| `0101` | schema.fields | list(reference Field) |
| `0102` | schema.constraints | list(reference Constraint) |
| `0110` | field.name | bytes |
| `0111` | field.value_shape | record ValueShape |
| `0112` | field.cardinality | unsigned (`0` one, `1` optional, `2` many) |
| `0113` | field.since_version | unsigned |
| `0120` | module.name | bytes |
| `0121` | module.imports | list(reference Import) |
| `0122` | module.exports | list(reference) |
| `0123` | module.required_effects | list(reference Effect) |
| `0130` | import.module | reference Module |
| `0131` | import.minimum_revision | bytes (exactly one revision identity) |
| `0140` | constraint.rule | reference DiagnosticRule |
| `0141` | constraint.operation | bytes |
| `0150` | effect.name | bytes |
| `0151` | effect.capability | reference Capability |
| `0160` | capability.name | bytes |
| `0170` | provenance.actor | bytes |
| `0171` | provenance.parent_revision | bytes (exactly one revision identity) |
| `0180` | refinement.from | reference |
| `0181` | refinement.to | reference |
| `0182` | refinement.evidence | bytes |
| `0190` | diagnostic.name | bytes |
| `0191` | diagnostic.arguments | list(record ValueShape) |
| `01a0` | diagnostic.rule | reference DiagnosticRule |
| `01a1` | diagnostic.revision | bytes (exactly one revision identity) |
| `01a2` | diagnostic.entity | bytes (empty or exactly one entity identity) |
| `01a3` | diagnostic.path | list(record PathSegment) |
| `01a4` | diagnostic.arguments | list(value) |
| `01a5` | diagnostic.severity | unsigned |
| `01b0` | diagnostic_report.status | unsigned |
| `01b1` | diagnostic_report.diagnostics | list(reference Diagnostic) |

Every field identity is the 14-byte zero prefix followed by the listed suffix.
Names are descriptive data for projections and diagnostics; identity, not name,
defines a field.

The complete diagnostic and path contracts are defined in
[`diagnostics-v1.md`](diagnostics-v1.md).

## ValueShape record

Value shapes use a record rather than adding wire tags:

| Field identity | Meaning |
|---|---|
| `00000000000000000000000000002000` | kind: unsigned wire value tag |
| `00000000000000000000000000002001` | referenced schema: optional reference |
| `00000000000000000000000000002002` | element shape: optional nested record |

The initial meta-schema uses only wire kinds, schema references, and homogeneous
lists. Later modules can define richer sums, widths, ranges, text encodings, and
domain types without changing the wire kernel.

## Validation phases

1. Decode and canonicalize the envelope without interpreting schemas.
2. Locate the candidate kernel module and Schema entity by identity.
3. Validate the self-description using fixed bootstrap expectations for only
   the Schema and Field shapes;
4. validate all remaining meta-schema entities from those declarations;
5. validate imported modules from their declared schema versions;
6. preserve but do not semantically certify declarations from unavailable
   imports.

The fixed expectation in phase 3 is the trusted semantic seed. It contains two
entity shapes, not a catalogue of language mechanics.

The executable bootstrap slice currently covers Module, Schema, Field,
schema.fields, field.value_shape, and module.exports. The checked validator
activates those requirements only for the reserved Kernel module identity,
verifies all seven declarations and their schema relationships, and rejects an
incomplete reserved module. Remaining meta-schema declarations stay candidate
until their field shapes are encoded and validated.

## External-module proof

The first extension fixture will define a `Counter` schema in a distinct module:

```text
Counter
  value: unsigned
  step: unsigned
```

It must pass using only Schema, Field, Module, and Import data. The kernel binary
must remain byte-identical. A second fixture adds an unknown optional field to
Counter; an implementation that does not know that field must round-trip it
unchanged at the semantic-value level. This is the required proof that semantic
modules extend the graph rather than expanding the kernel.

The checked `external-counter.a0` fixture now proves the structural half of
this gate: its unknown optional field is accepted and survives a validating
byte-identical round trip without changing the Kernel validator. Semantic
schema/import validation remains required before the full extension gate passes.
