# Target Contract v1

Target contracts describe how an execution backend can realize semantic
requirements. They are an external semantic module: neither WebAssembly nor
any other target changes Kernel meaning.

The module identity is `0000000000000000000000000000c000`. Its initial schema
identities are:

| Identity | Entity |
|---|---|
| `0000000000000000000000000000c010` | TargetContract |
| `0000000000000000000000000000c011` | Requirement |
| `0000000000000000000000000000c012` | SupportRule |
| `0000000000000000000000000000c013` | Resolution |
| `0000000000000000000000000000c014` | ExecutionPlan |
| `0000000000000000000000000000c015` | Boundary |

## TargetContract fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000c100` | target name | bytes |
| `0000000000000000000000000000c101` | contract revision | unsigned |
| `0000000000000000000000000000c102` | support rules | ordered list(reference SupportRule) |

## Requirement fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000c110` | semantic construct | reference |
| `0000000000000000000000000000c111` | minimum mechanic revision | unsigned |
| `0000000000000000000000000000c112` | required properties | ordered list(reference) |

A requirement names semantics, not syntax or a source language. Two projections
that produce the same requirement therefore receive the same target decision.

## SupportRule fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000c120` | semantic construct | reference |
| `0000000000000000000000000000c121` | maximum mechanic revision | unsigned |
| `0000000000000000000000000000c122` | preserved properties | ordered list(reference) |
| `0000000000000000000000000000c123` | realization | unsigned Fidelity value |
| `0000000000000000000000000000c124` | dependency | optional reference |
| `0000000000000000000000000000c125` | evidence | ordered list(reference) |

## Resolution fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000c130` | requirement | reference Requirement |
| `0000000000000000000000000000c131` | target contract | reference TargetContract |
| `0000000000000000000000000000c132` | selected rule | optional reference SupportRule |
| `0000000000000000000000000000c133` | realization | unsigned Fidelity value |
| `0000000000000000000000000000c134` | unresolved properties | ordered list(reference) |
| `0000000000000000000000000000c135` | diagnostics | ordered list(reference Diagnostic) |

## ExecutionPlan and Boundary fields

| Identity | Meaning | Shape |
|---|---|---|
| `0000000000000000000000000000c140` | root package/program | reference |
| `0000000000000000000000000000c141` | target contract | reference TargetContract |
| `0000000000000000000000000000c142` | resolutions | ordered list(reference Resolution) |
| `0000000000000000000000000000c143` | boundaries | ordered list(reference Boundary) |
| `0000000000000000000000000000c144` | executable | boolean |
| `0000000000000000000000000000c150` | boundary provider | reference |
| `0000000000000000000000000000c151` | boundary consumer | reference |
| `0000000000000000000000000000c152` | typed interface | reference |
| `0000000000000000000000000000c153` | transport/ABI | reference |

The solver analyzes the transitive package closure, not just isolated language
constructs. Every package assumption becomes a requirement: effects, ABI,
memory, concurrency, reflection, dynamic loading, filesystem, networking,
process access, architecture dependencies, and runtime services are explicit.
The resulting ExecutionPlan can mix strategies. A native island or embedded
runtime is therefore a visible region connected through typed Boundary
entities, not a false claim that the region was compiled exactly to Wasm.

## Fidelity values

The encoded values are stable for this module revision:

| Value | Meaning |
|---:|---|
| `0` | exact |
| `1` | refined |
| `2` | adapted |
| `3` | emulated |
| `4` | embedded runtime |
| `5` | native island |
| `6` | impossible |

`guarded` is a property of a selected realization, not a substitute for its
realization. A guard must be represented as a required dependency/effect and
reported in the resolution. The earlier architecture term `native` is
specialized here to `native island`: execution deliberately crosses out of the
selected target while retaining an explicit semantic boundary.

## Deterministic resolution

For one requirement and target contract, a resolver:

1. considers only rules for the same semantic construct whose supported
   revision includes the requirement revision;
2. rejects a rule if it does not preserve every required property;
3. orders remaining rules by fidelity value, then dependency identity, then
   rule identity;
4. selects the first permitted realization;
5. emits `impossible` with structured diagnostics if no rule remains.

The compatibility solver resolves every requirement in dependency order,
records generated boundaries and dependencies, then marks the plan executable
only when every resolution is permitted by policy. An `impossible` resolution
blocks an all-target plan. Policy may instead authorize a native island, but
that produces a mixed deployment and must be reported as such.

Policy may forbid adaptations, runtimes, or native islands, but cannot relabel
them. Dependencies and evidence are part of the result. A backend must never
silently drop a property or claim `exact` based only on successful execution.

## WebAssembly contract

WebAssembly is the first required serious target contract, not the definition
of Seme semantics. Its rules may cite core Wasm, WASI, the Component Model,
host adapters, embedded runtimes, or native islands as explicit dependencies.
The contract must distinguish those realizations; the presence of a Wasm
artifact alone is not evidence of exactness.

Conformance requires at least one fixture for every Fidelity value and must
show that an ordinary, non-Wasm-authored program resolves without source
catering when its requirements can be preserved. Impossible requirements must
produce semantic locations and reasons. The same canonical program remains
eligible for native AOT, interpreter, VM, JIT, and generated-source contracts.

The first real portability slice must ingest an ordinary package that was not
authored for Wasm, discover its package and runtime assumptions, emit the plan,
produce a Wasm component where permitted, and compare behavior with the
package's original toolchain. Source changes made only to appease Wasm fail the
continuity proof.

## First canonical planning proof

The canonical module and first deterministic resolver are implemented for a
scoped ordinary Go profile. The fixture calls `log.Printf` after computing a
quota decision. The provider uses Go parsing and type information to prove the
exact call target, fixed message, logged value, return value, dependency, and
effect. It emits:

- package dependency `go:log`;
- Foundation effect and capability `observability.log`;
- target `wasm32-pulp-v1`;
- an `adapted` rule requiring `pulp.host.log-v1`;
- an explicit typed Boundary to that host import;
- canonical evidence and runtime assumptions.

With `allow-adapted` policy the plan is executable and retains fidelity `2`.
With `exact-only` policy the same rules are not relabeled: both requirements
resolve as `impossible` and the plan is non-executable. Native source changes,
unsupported message semantics, and invented policies reject.

This completes canonical target planning. The subsequent scoped Wasm v1 backend
now consumes the checked executable plan and rejects the exact-only plan; actual
Pulp execution remains the next runtime proof.
