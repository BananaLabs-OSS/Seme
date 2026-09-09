# Useful Application Bridge Profile v1

## Purpose

This specification gives `100%` a finite, reproducible meaning for Seme's Go,
JavaScript, and Lua bridges. It does not define complete support for any of
those languages.

Seme may claim **Useful Application Bridge Profile v1 complete** only when all
required capabilities and all three ecosystem columns below pass their gates.
The denominator is frozen by this document. Later capability growth creates a
new profile version instead of changing the meaning of an old percentage.

The profile begins after the Core v30 restoration sequence. Core checkpoints
prove individual semantic families; this profile proves that those families
compose in ordinary programs.

## Product boundary

A conforming application is a deterministic command-processing module with
typed state, validation, collection updates, dynamic behavior selection,
callbacks, errors, and declared effects. It is large enough to implement a
small service, rules engine, document transformer, command-line core, or game
rules module. User interfaces, operating-system integration, arbitrary package
compatibility, networking stacks, concurrency, reflection, and foreign
interfaces are outside v1.

Humans author and receive ordinary native source. Go, JavaScript, and Lua each
connect directly to canonical Seme; no language is translated through another
language.

## Completion matrix

Every row is mandatory. A cell passes only when the ecosystem's native source
is accepted by its provider, produces the expected canonical meaning, executes
natively, projects back to native source, and re-lifts to equivalent canonical
meaning. `N/A` is not permitted in v1.

| ID | Composed capability | Go | JavaScript | Lua |
|---|---|---:|---:|---:|
| UAB-01 | Modules, named functions, typed parameters, calls, and multiple source files | required | required | required |
| UAB-02 | Signed i64, Boolean, text, bytes, records, result/option values, arrays, slices, and runtime-keyed maps | required | required | required |
| UAB-03 | Locals, assignment, arithmetic, comparison, short-circuit logic, conditionals, loops, and early return | required | required | required |
| UAB-04 | Collection construction, length, indexing, traversal, update, append, lookup, insert, and removal | required | required | required |
| UAB-05 | Methods or native equivalent, interfaces/protocol tables, and bounded dynamic dispatch | required | required | required |
| UAB-06 | Immutable closures and mutable captured environments | required | required | required |
| UAB-07 | Explicit state transition returning updated state plus result | required | required | required |
| UAB-08 | Declared fallible operations and explicit error propagation | required | required | required |
| UAB-09 | Capability-authorized effect request with deterministic effect trace | required | required | required |
| UAB-10 | Stable semantic identity through formatting, rename, projection, and re-import | required | required | required |
| UAB-11 | One cumulative stateful application using UAB-01 through UAB-10 together | required | required | required |
| UAB-12 | Cross-language equivalence: all three projections re-lift to the same canonical application | required | required | required |

The frozen machine-readable denominator and current evidence claims live in
`conformance/uab-v1/scorecard.json`. Run `./scripts/check-uab-v1-scorecard.sh`
to validate its exact 36-cell shape and compute the per-language and overall
scores. Evidence names may be added to a cell only by a gate that directly
produces that evidence.

## Required application proof

The cumulative fixture is not a collection of one-feature functions. It must
accept a command and immutable state, select an implementation through bounded
dynamic dispatch, traverse and update runtime collections, invoke a captured
callback, and return an explicit transition containing either a new state and
ordered effects or a typed rejection. At minimum it must exercise:

- two source modules and an internal call;
- one record containing text, a slice, and a runtime-keyed map;
- a loop containing a conditional and an early rejection;
- one interface/protocol implementation selected at runtime;
- one immutable capture and one mutable capture;
- successful, missing-key, invalid-index, and arithmetic-boundary cases;
- an authorized effect and rejection when its capability is absent; and
- at least 100 generated command sequences compared across all realizations.

The same canonical application must execute through the canonical interpreter,
the supported Wasm target, and pinned Pulp. Native Go, JavaScript, and Lua are
behavioral oracles, not runtime dependencies of canonical execution.

## Per-cell evidence gate

Each matrix cell has exactly five independently reported assertions:

1. **Lift:** real ecosystem parsing and analysis accepts the declared native
   construct and emits validated canonical semantics.
2. **Native parity:** boundary, adversarial, and generated vectors produce the
   same values, state, errors, and ordered effect traces natively and
   canonically.
3. **Target parity:** every claimed target produces those same observations.
4. **Projection round trip:** canonical semantics project to idiomatic,
   executable native source and re-lift to canonical equivalence.
5. **Rejection:** a nearby unsupported or semantically incompatible construct
   fails with a located diagnostic and never receives guessed meaning.

A cell scores `passed` only at 5/5. A language column scores `100%` only at
12/12 passed cells. The overall UAB-v1 score is passed cells divided by 36.
Failed, missing, and untested cells all contribute zero; partial internal work
may be shown as evidence counts but not as a passed cell.

## Ecosystem fidelity requirements

Native spelling does not establish semantic equivalence. Every mapping records
`exact`, `refined`, `adapted`, `emulated`, `embedded runtime`, `native island`,
`remote provider`, or `impossible` as defined by the language bridge roadmap.

- Go must use the Go parser, type checker, package loader, formatter, and native
  test toolchain. Value semantics, integer overflow, map observations, method
  sets, nil states, and error behavior must be classified explicitly.
- JavaScript must use a declared ECMAScript and Node profile. BigInt-backed i64
  is an adaptation, not JavaScript Number. UTF-16 observations, coercion,
  truthiness, `null`, `undefined`, prototypes, exceptions, and async behavior
  cannot be silently treated as Core behavior.
- Lua must use a declared Lua edition and implementation. Numeric model,
  table key equality and iteration, one-based indexing, multiple returns,
  metatables, nil deletion, lexical environments, and error behavior must be
  classified explicitly.

Projection quality is tested with ecosystem-native linters/formatters where
available and reviewable golden source. Canonical equality alone is
insufficient if a projection changes an ecosystem-observable behavior.

## Versioned broad-compatibility scores

“Broad Go/JavaScript/Lua compatibility” is not a single percentage and may not
use an informal estimate after this profile is adopted. Publish a scorecard for
each language and versioned environment:

```text
language edition + implementation/toolchain version + OS/architecture profile
```

Each scorecard freezes four separate denominators:

| Dimension | Denominator | Unit |
|---|---|---|
| Language | Enumerated normative language features for the declared edition | feature cases |
| Runtime | Enumerated observable runtime mechanics in the declared profile | mechanic cases |
| Standard library | Frozen list of public package/module operations selected for the profile | operation cases |
| Tooling/package system | Frozen build, module resolution, test, format, metadata, and foreign-boundary workflows | workflow cases |

An item passes only with lift, native parity, target parity where claimed,
projection/re-lift, and negative evidence. Report raw counts and percentages
for every dimension; never average them into one number. For example:

```text
Go Profile 1 (declared toolchain):
language 41/120; runtime 7/30; standard library 12/400; tooling 3/20
```

The example numbers are illustrative, not current results. Feature inventories
must cite the governing language/runtime documentation and be checked into the
repository. Adding items creates a new scorecard version so an old score remains
reproducible. Package popularity, lines of code accepted, and successful parsing
are supplemental measurements, never substitutes for semantic conformance.

No profile should be called “full” while any normative feature in its declared
edition is absent. Third-party ecosystem compatibility is reported separately
as an explicit package corpus with pinned revisions and tests. Because package
ecosystems evolve continuously, an unqualified claim of 100% broad ecosystem
compatibility is not a finite milestone.

## Current baseline audit

At Core v30, repository evidence is strongest for a bounded shared subset:
typed functions; i64/Boolean/text and selected record/result/collection values;
locals, branches, loops, calls, effects, fixed arrays and i64 slices; collection
queries and immutable updates; methods; explicit transitions; and bounded
interface dispatch with explicit satisfaction witnesses; and immutable lexical
closures with explicit environments; and explicitly state-threaded mutable
closure environments; and runtime-keyed maps composed through collection
folds. Go and JavaScript have native parity and projection evidence for
selected shapes. A direct bounded Lua provider and projector now cover the
UAB-01 Boolean function/call slice without using either language as an
intermediate representation.

The repository justifies all 36/36 UAB-v1 cells. UAB-01 through UAB-11 pass
all five evidence classes independently for Go, JavaScript, and Lua. UAB-12
selects one canonical application, projects it to all three languages, runs
all 2,048 generated observations natively and through the canonical, Wasm,
and pinned-Pulp realizations, and requires every projection to re-lift to the
same canonical bytes. The scope remains bounded because:

- the Core checkpoint sequence and both cumulative architectural composition
  gates are complete, establishing the prerequisite for this profile;
- every language bridge remains deliberately bounded to the capabilities
  evidenced by this profile rather than claiming its full ecosystem.

The machine-readable scorecard records all 36 cells and reports only cells
whose complete gates pass. Historical estimates remain separate and must not
be converted into profile scores.

## Delivery sequence after Core v30

1. Freeze machine-readable UAB-v1 cases and expected canonical observations.
2. Generalize Go lifting/projection until its 12 cells pass compositionally.
3. Generalize JavaScript lifting/projection against the same cases while
   preserving JavaScript-specific fidelity classifications.
4. Implement Lua provider and projection directly against canonical Seme, then
   pass the same 12 cells using a pinned Lua runtime.
5. Run the cumulative application through native oracles, canonical execution,
   Wasm, and Pulp; publish the 36-cell report.
6. Freeze initial broad-compatibility inventories for Go, ECMAScript/Node, and
   Lua, then grow those independently according to demonstrated user needs.

## Completion claim

When all gates pass, the accurate statement is:

> Seme supports 100% of Useful Application Bridge Profile v1 in Go,
> JavaScript, and Lua, with equivalent canonical meaning and declared fidelity.

It is inaccurate to shorten that statement to “Seme supports 100% of Go,
JavaScript, and Lua.”
