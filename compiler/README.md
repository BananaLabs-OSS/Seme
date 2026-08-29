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
imported references, or perform schema-level migrations.

`kernel-wire-validator-diagnostic.seme` is the post-Frozen-Core successor. With
one argument it validates exactly like the frozen validator. With a result path
it writes the unchanged canonical input on success or a canonical
DiagnosticReport on rejection. Its stable `kernel.invalid_input` rule uses kind
`0` for invalid decoded graphs and kind `1` for malformed encodings. The frozen
validator remains checked separately so Frozen Core 1 stays reproducible.

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
semantic module and derives a portable K0 image. Bootstrap profile 6 accepts
multiple P1/P2 functions with arbitrary valid parameter/local counts and
arbitrary-length instruction lists containing constants, locals, stack/integer
operations, comparisons, and branches. It checks declared entity counts, K0
resource bounds, module schemas, field shapes, and every semantic reference it
consumes. The bootstrap profile groups each Function immediately before its
Instruction entities; the general identity-table reader will remove that
storage-order constraint.

The checked `return-42.seme` graph lowers to `return-42.k0`, which the frozen K0
executor runs with result 42. Function indices, instruction indices, code size,
and byte layout exist only in the derived image. This is the first complete
canonical-semantics-to-execution path. A second checked graph derives `20 + 22`
through separate instruction entities and also executes as 42. A third graph
stores and reloads 42 through an explicit local, guarding the distinction
between Program capabilities and Function parameter/local counts. The backward
loop graph uses a stable instruction reference, from which the lowerer derives
both the K0 target index and byte witness, loops three times, and returns 42.
The forward-branch graph proves the corresponding forward reference using four
monotone layout refinements before emission. The checked
two-function graph proves ordered function derivation and entry selection by a
stable reference; the second function is selected and returns 42.
The call graph passes two semantic arguments to a separately identified
function and derives its K0 call index, returning `20 + 22 = 42`.
The P2 fixtures prove buffer operations and capability-scoped compiler I/O. A
canonical graph reads one argument path and writes the same bytes to another;
the emitted artifact declares exactly the argument/read/write capability mask.

## Canonical lowerer self-reproduction

`k0-module-lowerer.g1` is the readable projection of the compiler graph and
`k0-module-lowerer.seme` is its canonical source. The graph was deterministically
lifted once from the human-authored S1 bootstrap compiler: K0 function and
instruction structure became semantic entities, derived branch witnesses and
numeric call indices became stable references, and no executable bytes were
embedded as data. The temporary lifter is not part of this repository or the
reproduction path.

The checked lowerer A compiles the canonical graph to B; B compiles it to C.
A, B, and C are byte-identical. Canonical Seme is therefore authoritative for
this lowerer now; `k0-module-lowerer.s1` remains as auditable bootstrap history,
not the continuing source required to reproduce it.

The G1 frontend, Kernel validator, and v0 migrator now follow the same model:
each has a checked readable `.g1` projection, authoritative canonical `.seme`
graph, and derived `.k0` executable. Compiler C rebuilds all three executables
byte-for-byte. Frozen Core 1 records the closed reproduction and continuing-edit
contract in `FROZEN-CORE-1.md`.

Branch and call lowering retains semantic references in canonical storage.
Like the symbolic S1 compiler, the canonical lowerer derives positional indices
and branch witnesses through monotone layout refinement before emission. The K0
module contract now distinguishes an instruction `target` reference from a
function `callee` reference; neither is serialized canonically as a derived
numeric index.
