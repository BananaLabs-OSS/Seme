# Located Diagnostics 1

Located Diagnostics 1 is the completed structured-diagnostic checkpoint built
on Frozen Core 1. The canonical implementation is
`kernel-wire-validator-located.g1`; its serialized graph and executable lowering
are `kernel-wire-validator-located.seme` and
`kernel-wire-validator-located.k0`.

## Guarantee

The validator preserves valid canonical Kernel wire v1 inputs byte-for-byte.
Every currently covered invalid input exits with status 65 and, when given a
result path, writes a canonical `DiagnosticReport` that the frozen validator
accepts. Reports classify decoded semantic rejection as kind 0 and malformed
encoding as kind 1.

When trustworthy, reports carry the input revision and entity identities. A
field-level failure contains a field path segment. Its all-zero field identity
is the specified sentinel meaning that this validator generation reached the
field semantic level but did not certify a particular field identity. Exact
field identity and nested/list/reference provenance are later refinements, not
claims of this checkpoint.

No source projection, host compiler, Python runtime, Go toolchain, or external
language runtime is required by the committed reproduction path.

## Reproduction

From the repository root:

```text
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 \
  compiler/kernel-wire-validator-located.g1 rebuilt.seme

bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator.k0 \
  rebuilt.seme validated.seme

bootstrap/seme-k0-linux-amd64 compiler/k0-module-lowerer.k0 \
  rebuilt.seme rebuilt.k0

cmp rebuilt.k0 compiler/kernel-wire-validator-located.k0
```

The final comparison must be byte-exact.

## Conformance gate

The checkpoint was tested with five valid fixtures (`minimal`, `values`,
`forward-reference`, `external-counter`, and `kernel-meta-minimal`) and seven
invalid fixtures (`kernel-meta-incomplete`, `dangling-reference`,
`duplicate-entities`, `trailing-byte`, `unknown-value-tag`, `unsorted-parents`,
and `noncanonical-version`). Valid outputs were byte-exact. Every invalid input
returned 65 and produced a canonical report.

`conformance/diagnostics/unknown-value-tag.g1` freezes the field-level located
report shape and its explicit unavailable-identity sentinel.
