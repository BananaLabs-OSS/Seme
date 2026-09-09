# Useful Project Bridge Profile v1

## Purpose

Useful Project Bridge v1 (`seme.upb/v1`) is a finite project-level conformance
profile above the Useful Application Bridge. It asks whether a bounded ordinary
project can be discovered, represented, executed, projected, revised, and
realized without confusing source-language tooling with canonical meaning.

UPB-v1 does not expand the Kernel and does not place a downstream application
inside Seme. Consumer projects remain external and own their own pinned
acceptance harnesses. Workbench is one possible later consumer; it is not a
fixture, special case, implementation dependency, or completion criterion for
this profile.

## Prerequisite and responsibility boundaries

Every language column requires its complete `seme.uab/v1` column. UPB reuses:

- Package Contract v1 for typed interfaces, dependencies, effects, runtime
  assumptions, and fidelity;
- Provider Contract v1 for provenance, opaque regions, projection, identity,
  and reconciliation;
- Target Contract v1 for requirements, realization fidelity, boundaries, and
  execution plans;
- Live Language Service v1 for ordered snapshots and last-valid revisions;
- Pure Value ABI and Application Wire contracts for bounded values; and
- Pulp Target v1 for execution and capability enforcement.

A versioned Project Contract above Package Contract may add neutral project
records such as project snapshots, source-unit classifications, package
membership, dependency locks, build profiles, resource identities,
configuration bindings, lifecycle plans, and project revisions. Package
manager, storage, transport, and operating-system mechanics remain provider or
target concerns. They do not become Core semantics merely because a fixture
uses them.

## Matrix

The denominator is 12 capabilities for each of Go, JavaScript, and Lua: 36
cells. A cell passes only when all seven evidence classes are present and its
authoritative gate passes.

| ID | Bounded project capability | Go | JavaScript | Lua |
|---|---|---|---|---|
| UPB-01 | Deterministic project snapshot and source classification | required | required | required |
| UPB-02 | Package/module graph, visibility, imports, and exports | required | required | required |
| UPB-03 | Resolved and integrity-pinned dependency closure | required | required | required |
| UPB-04 | Typed cross-package calls and values | required | required | required |
| UPB-05 | Typed configuration and deterministic initialization | required | required | required |
| UPB-06 | Digest-addressed resources and assets | required | required | required |
| UPB-07 | Versioned durable-state port and one migration | required | required | required |
| UPB-08 | Bounded request/event transport, ordering, and replay | required | required | required |
| UPB-09 | Declared clock, seeded randomness, and external effects | required | required | required |
| UPB-10 | Deterministic build, target resolution, and placement plan | required | required | required |
| UPB-11 | Project-scale incremental revision and reconciliation | required | required | required |
| UPB-12 | One cumulative useful project and exact cross-language projection | required | required | required |

## Seven evidence classes

Each matrix cell has exactly these assertions:

1. **Native project:** the ordinary project is discovered and succeeds through
   its declared native build and test commands before import.
2. **Project lift:** the complete declared closure lifts into a validated
   canonical project graph with deterministic identities and provenance.
3. **Canonical parity:** native and canonical executions agree on values,
   state, typed errors, and ordered effect traces over boundary, adversarial,
   and generated cases.
4. **Resolution fidelity:** dependencies, runtime assumptions, capabilities,
   and every realization boundary resolve with explicit fidelity and evidence.
5. **Target parity:** every target claimed by the cell produces the same
   observations; unsupported targets resolve as adapted, mixed, or impossible
   rather than being silently relabeled exact.
6. **Project round trip:** projection produces an ordinary independently
   buildable native project whose re-import is canonically equivalent and whose
   untouched and opaque regions satisfy their preservation contracts.
7. **Atomic rejection:** nearby unsupported, stale, conflicted, tampered,
   undeclared, malformed, or over-limit project states reject without partial
   graph, source, artifact, state, or effect commits.

Evidence is independently reported. Missing, failing, or unexecuted evidence
contributes zero. A language column is 100% only at 12/12 passing cells; the
overall score is passing cells divided by 36.

## Capability requirements

### UPB-01 — deterministic project snapshot

Record project root identity, relative source paths, content digests, declared
toolchain/profile, and explicit tracked, ignored, generated, vendored, and
opaque classifications. Repeated discovery must be byte-deterministic.
Traversal outside the root, classification ambiguity, and digest drift reject.

### UPB-02 — package/module graph

Represent multiple packages or modules, stable membership, imports, exports,
and native visibility. Missing imports, duplicate identities, illegal cycles,
and inaccessible declarations reject with project locations.

### UPB-03 — dependency closure

Resolve a bounded local dependency and a pinned ecosystem dependency. Record
version, integrity, source, and applicable metadata; prove an offline repeated
resolution. Floating, substituted, undeclared, or integrity-mismatched
dependencies reject. This cell does not imply general registry compatibility.

### UPB-04 — typed cross-package behavior

Compose UAB-v1 values and calls through real package boundaries without
flattening packages or translating through another source language. Native,
canonical, projected, and target observations must agree.

### UPB-05 — configuration and initialization

Lift typed configuration, defaults, validation, explicit initialization order,
and lifecycle state. Ambient environment reads have no implicit neutral
meaning; they require declared configuration or capability contracts.

### UPB-06 — resources and assets

Declare bounded text and byte resources by stable identity, path, media kind,
size, and digest. Lookup and projection/copy rules must preserve bytes.
Traversal, missing resources, changed digests, duplicate destinations, and
over-limit content reject.

### UPB-07 — durable state

Use a capability-backed storage port with a versioned schema, atomic
load/update/save, and exactly one declared migration. Partial writes,
incompatible versions, invalid migrations, and unauthorized storage reject.
Filesystem or database mechanics remain target/provider realizations.

### UPB-08 — request/event transport

Use one bounded typed command/event stream with deterministic ordering,
message-size limits, correlation identity, replay, and explicit
duplicate/out-of-order behavior. Malformed frames and violated ordering reject
without partial state or effects. This does not claim arbitrary protocols.

### UPB-09 — nondeterminism and effects

Inject a declared clock and seeded random source and request one external
effect. Replay must reproduce state and ordered traces. Missing capabilities
reject before observable work; ambient time and randomness remain unsupported.

### UPB-10 — build and placement

Derive requirements from the transitive canonical closure and emit a
deterministic target/placement plan. Exact, refined, adapted, emulated,
embedded-runtime, native-island, and impossible regions remain visible. The
same inputs produce byte-identical permitted artifacts.

### UPB-11 — live project revision

Process complete multi-file snapshots with monotonic revisions and last-valid
retention. Apply one identity-bound semantic edit spanning package references,
project atomically, run native validation, and re-import. Stale, both-changed,
ambiguous, and conflicted revisions reject without partial writes.

### UPB-12 — cumulative useful project

The cumulative proof is one modest deterministic command service or CLI with at
least three packages: domain behavior, a durable-state port, and a transport
adapter. It includes typed configuration, one resource, explicit state/result,
a capability-authorized effect, injected time/randomness, and replay.

The project is represented canonically once and projected independently to Go,
JavaScript, and Lua. Every projection must build and test natively and re-lift
to the same canonical bytes. Native, canonical, supported Wasm, and pinned Pulp
realizations must agree over at least 4,096 generated command observations.
Dependency, resource, revision, capability, boundary-value, and graph
tampering must reject atomically. A mixed-language deployment is useful
additional evidence but cannot replace any independent language column.

## Completion gate

The profile is complete only when a checked mapping associates every claimed
cell with an authoritative top-level gate, a clean complete-profile runner
executes each mapped gate without recursive score self-certification, UPB-12
passes its shared proof, and the machine-readable score reports 36/36.
Shape-only score validation never establishes evidence truth.

Removing Seme-specific evidence from each projected fixture must leave an
ordinary native project that retains its declared behavior. External consumer
harnesses may subsequently demonstrate usefulness without changing this
profile or introducing consumer-specific branches into Seme.

## Anti-overclaim boundary

UPB-v1 proves only the declared language/toolchain editions, finite fixture
closures, pinned dependencies, one storage schema and migration, one transport,
injected clock and random source, declared effects, supported targets, and the
frozen generated corpus.

It does **not** prove arbitrary Go, Node, or Lua projects; general npm, Go
module, or LuaRocks compatibility; arbitrary dependency conversion; native
FFI; threads or data-race semantics; reflection or metaprogramming; dynamic
loading; operating-system, UI, GPU, editor, or game-engine behavior; arbitrary
databases or network protocols; lossless projection of arbitrary source; or
universal language/runtime interoperability. Opaque, generated, vendored,
embedded-runtime, remote, and native-island regions stay explicit and do not
count as lifted semantics.

