# Go UPB-10 acceptance design

Status: implemented and claimed by `scripts/check-go-upb-10.sh` on 2026-09-11.

Go UPB-10 extends the cumulative Project-v12 fixture with deterministic target
resolution and physical placement derived from its authenticated transitive
closure. It reuses Target Contract v1. It does not add Go, WebAssembly, Pulp,
an operating system, or a downstream product to Core semantics.

## Authority and project binding

Project Contract v13 adds one neutral `PlacedProjectSnapshot` that binds:

- the exact Project-v12 `EffectProjectSnapshot`;
- one exact Target Contract v1 `ExecutionPlan` root;
- the target-contract identity and revision used to resolve it; and
- a content-derived project revision.

The target plan remains an independently valid semantic artifact. Project v13
does not copy target mechanics into the Project schema, and it cannot make an
invalid or non-executable plan executable.

## Derived closure

The provider derives requirements from authenticated Project-v12 authority,
not from filenames, a hand-maintained fixture list, or product knowledge. The
bounded closure includes:

- the canonical command planner and Pure Value ABI;
- owned package calls and the pinned dependency closure;
- typed configuration and initialization order;
- detached resource identities and placements;
- the durable-state Load/compare-exchange port;
- ordered request/event framing and its port;
- explicit clock observations and seeded random transitions;
- the ordered external Boolean effect request; and
- deterministic replay without live capabilities.

Every derived requirement retains its originating semantic identity, minimum
mechanic revision, required properties, owner, and evidence. A missing,
duplicated, unreachable, unauthenticated, or source-only requirement rejects.

## Resolution and placement

Resolution follows Target Contract v1 ordering exactly. Fidelity remains one
of `exact`, `refined`, `adapted`, `emulated`, `embedded runtime`, `native
island`, or `impossible`; policy may reject a class but cannot rename it.

The first selected mixed plan places the capability-free planner, seeded
random transition, replay, and supported ABI in the pinned Pulp/Wasm execution
island. Live clock sampling, external effect delivery, DurablePort, and
TransportPort remain authenticated Go-host native islands because the pinned
Pulp revision has no conforming providers. Typed boundaries name provider,
consumer, interface, transport/ABI, capability, and evidence.

An all-Wasm/exact-only policy resolves those unavailable host mechanics as
`impossible` and marks the plan non-executable. The plan must not silently use
ambient WASI clocks, host entropy, filesystem storage, SQLite, synchronous
opaque calls as streams, or logging capture as proof of external delivery.

The resolver also accepts independently authenticated rule fixtures for every
Target v1 fidelity value and reproduces each value unchanged. Those fixtures
prove enum preservation; they do not claim that the cumulative project needs
every mechanism or that every plan is executable.

## Determinism and artifacts

Equal Project-v12 authority, target contract, provider catalog, policy, and
toolchain pins produce byte-identical:

- requirement closure;
- resolutions and boundaries;
- execution plan;
- Project-v13 artifact;
- source-free report;
- permitted Wasm/Pulp artifact set; and
- native-island launch manifest.

Changing policy, provider evidence, target revision, package/dependency
authority, capabilities, ABI, or a reachable semantic requirement must change
the appropriate digest. Source comments and filenames that leave canonical
meaning and authenticated presentation unchanged must not change target
resolution.

## Seven evidence classes

1. **Native project.** The complete ordinary Go project retains its race tests
   and 4,096-observation controlled-effects corpus before planning.
2. **Project lift.** Two lifts derive byte-identical requirement closures,
   plans, Project-v13 artifacts, and closed-world bundles.
3. **Canonical parity.** The planned pure execution island retains exact native
   and canonical state/result/effect observations.
4. **Resolution fidelity.** Every transitive requirement is accounted for;
   every fidelity value remains unchanged; dependencies, evidence, policy, and
   typed boundaries are authenticated.
5. **Target parity.** Permitted standalone Wasm and pinned Pulp artifacts retain
   the UPB-09 4,096-case parity. Unsupported host mechanics stay visible as
   native islands or impossible resolutions.
6. **Project round trip.** Source-free projection builds offline and re-lifts
   to exact canonical meaning; the same authority regenerates the same plan,
   while honest source provenance may revise Project-v13.
7. **Atomic rejection.** Every adversary rejects without a partial plan,
   Project-v13 artifact, bundle, source tree, target artifact, or launch
   manifest.

## Required adversaries

The cumulative gate rejects:

- an omitted, duplicated, invented, unreachable, or source-only requirement;
- a requirement whose construct, revision, property, owner, or evidence was
  changed;
- a support rule for the wrong construct/revision or missing a property;
- fidelity relabeling, including native-island or adapted behavior called
  exact;
- nondeterministic rule ordering or ambiguous equal-ranked rules;
- a forbidden adaptation, emulation, embedded runtime, or native island;
- a selected rule with missing/substituted dependency or evidence;
- a missing, malformed, or untyped boundary;
- ambient clock/entropy or false DurablePort/TransportPort substitutions;
- an executable plan containing an impossible resolution;
- a non-executable plan used to publish target artifacts;
- stale or mixed Project-v12, target contract, provider catalog, policy, plan,
  Project-v13, target artifact, or launch manifest;
- symlinked, over-limit, existing, or colliding output destinations; and
- any failure path that leaves partial observable output.

## Claim rule

Only `scripts/check-go-upb-10.sh` may claim or map Go UPB-10. It executes
the complete mapped UPB-09 gate, the independent Target Contract v1 gate, the
new Project-v13 and placement partitions, all seven evidence classes, and a
clean deterministic second build. The cumulative gate passes; the scorecard is
10/36 overall and 10/12 for Go.
