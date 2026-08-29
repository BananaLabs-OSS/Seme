# Architecture

## Product definition

Seme separates five concerns that conventional programming languages commonly
bundle together:

1. presentation;
2. semantics;
3. mechanics;
4. packages and runtimes;
5. execution targets.

Users edit one canonical program through textual, visual, conversational, or
direct semantic interfaces. A projection may resemble an existing language
without silently adopting that language's mechanics.

## Developer continuity contract

Seme is a universal semantic joiner, not a mandatory replacement user
experience. Adopting Seme in an existing project must not require its developer
to abandon the project's normal source files, directory layout, editor,
debugger, package manager, test commands, framework, engine, version-control
workflow, or deployment targets. A new Seme-connected project may likewise
present exactly like an ordinary project in its chosen ecosystem.

Native project files are writable projections and interoperability boundaries.
Seme may import them into canonical semantic identities and project accepted
changes back, but it must preserve constructs it cannot yet understand rather
than silently reinterpret or discard them. Adoption must be incremental down
to a file or explicitly delimited region. A project must remain usable by its
native tools and must be able to leave Seme without losing source or semantics
that Seme claimed to preserve.

G1, Kernel wire, bootstrap profiles, and canonical storage are normally
invisible implementation layers. No developer is required to learn them merely
to use a Seme-connected package or project.

WebAssembly is a useful analogy but a different boundary: WebAssembly joins
execution producers and runtimes through a portable machine format; Seme joins
program meaning, mechanics, ecosystems, presentations, and targets through a
canonical semantic model. WebAssembly can be one Seme target or package bridge.

Every native import/projection must report its fidelity. An `exact` round trip
requires conformance evidence covering native parsing, unchanged-region
preservation, semantic edits, generated native output, and behavior under the
original ecosystem toolchain. Anything weaker is labeled explicitly and may
remain an opaque native region.

## Kernel and semantic-foundation boundary

Frozen Kernel v1 is limited to:

- stable identity and revision;
- declarations, references, and structural values;
- versioned module and schema records;
- structural references, holes, and canonical serialization.

Versioned semantic-foundation modules define constraints, effects,
capabilities, provenance, refinements, and structured diagnostics. Seme owns
those semantics without hard-coding them into the frozen wire decoder.

Memory models, concurrency, relational semantics, object systems, ECS, UI,
machine semantics, package ecosystems, and surface grammars are modules—not
kernel concepts.

## Fidelity

Fidelity describes a particular mapping edge between a source semantic entity
and a destination realization:

- `exact`
- `refined`
- `emulated`
- `guarded`
- `adapted`
- `embedded`
- `native`
- `impossible`

A mapping records required decisions, dependencies, guards, evidence, and
diagnostics. Fidelity is never inferred merely from presentation syntax.

The machine-readable contract and deterministic selection rules are defined by
the external [`target-contract-v1.md`](target-contract-v1.md) module. WebAssembly
is its first mandatory serious conformance target, but is not a Kernel concept.

## Canonical state

Canonical state is semantic entities, stable identities, relationships,
mechanics, provenance, holes, and revisions. Generated source and displayed
syntax are disposable projections.

## Independence

The trusted reproduction path may assume only a documented processor ISA,
platform ABI, frozen seed bytes, and versioned Seme specifications. External
toolchains may be used as optional frontends, backends, package providers,
development conveniences, and differential oracles.

Self-hosting and universality are separate claims. Self-hosting proves that Seme
implements and reproduces its own compiler. Language or mechanic support
requires separate conformance evidence.
