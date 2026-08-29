# Compiler

This directory is reserved for the human-authored semantic Seme compiler.

The compiler will be implemented above the frozen bootstrap chain. Its source
of truth will be canonical Kernel v1 entities and modules, not generated native
code or the temporary S0 presentation. Until that representation is frozen,
bootstrap compiler sources remain under `bootstrap/`.

The handoff criterion is strict: the semantic compiler must rebuild itself,
preserve conformance behavior and diagnostics, and let subsequent platform work
proceed without modifying assembly.

## Kernel wire validator

`kernel-wire-validator.s1` is the first compiler-layer program authored above
the self-hosted bootstrap. Checked S1 compiles it deterministically to
`kernel-wire-validator.k0`.

The current structural profile validates the Kernel v1 envelope, canonical
ULEB integers, bounds, strictly ordered parent/entity/field identities,
recursive lists and records, known value tags, nesting depth, and exact
end-of-file. With an output path it writes the validated canonical envelope
byte-for-byte, providing the preservation path for semantics the current
validator does not interpret. A second append-only pass resolves local forward
references against the complete entity table and rejects dangling references.
For the reserved Kernel module identity it additionally requires the finite
Module/Schema/Field bootstrap declarations and verifies their schema
relationships. It does not yet validate every declared field shape, resolve
imported references, perform schema-level migrations, or produce structured
diagnostics.

`kernel-migrate-v0.s1` is a deterministic migration from the deliberately
obsolete v0 envelope, which lacked parent-revision metadata, to canonical v1.
The v1 validator rejects the old fixture, accepts the migrated graph, and the
migration preserves the entity payload while inserting a zero parent count.

## G1 readable graph projection

`g1-compiler.s1` is the first readable authoring projection for complete Kernel
wire graphs. It is itself compiled deterministically by the checked S1 compiler.
G1 separates human-readable graph construction from canonical storage: the G1
emitter produces Kernel wire v1 and the independent validator establishes that
the result is canonical.

`kernel-meta-minimal.g1` replaces the previous A0 byte construction as the
readable source for the finite Kernel meta-schema. Its checked canonical output,
`kernel-meta-bootstrap.seme`, is byte-identical to the A0-built envelope. The G1
Counter fixture is likewise byte-identical to its older A0 construction. These
equalities demonstrate a change in presentation without a change in canonical
semantic state.

## First semantic executable lowering

`k0-module-lowerer.s1` consumes a validated canonical Kernel graph using the K0
semantic module and derives a portable K0 image. Bootstrap profile 2 accepts one
P1 function with arbitrary parameter/local counts and an arbitrary-length list
of straight-line constant, local, stack, integer arithmetic, and comparison
instructions. It checks the module, schemas, field shapes, and every semantic
reference it consumes.

The checked `return-42.seme` graph lowers to `return-42.k0`, which the frozen K0
executor runs with result 42. Function indices, instruction indices, code size,
and byte layout exist only in the derived image. This is the first complete
canonical-semantics-to-execution path. A second checked graph derives `20 + 22`
through separate instruction entities and also executes as 42. A third graph
stores and reloads 42 through an explicit local, guarding the distinction
between Program capabilities and Function parameter/local counts. Branches,
calls, multiple functions, and P2 effects remain required before moving the
compiler graph onto it.

Branch and call lowering will retain semantic references in canonical storage.
Like the symbolic S1 compiler, the canonical lowerer derives positional indices
and branch witnesses through monotone layout refinement before emission. The K0
module contract now distinguishes an instruction `target` reference from a
function `callee` reference; neither is serialized canonically as a derived
numeric index.
