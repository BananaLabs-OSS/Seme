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

## Kernel boundary

Kernel v1 is limited to:

- stable identity and revision;
- declarations, references, and structural values;
- constraints and explicit holes;
- effects and capabilities;
- provenance;
- refinement relationships;
- versioned module and schema records;
- deterministic diagnostics and canonical serialization.

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

