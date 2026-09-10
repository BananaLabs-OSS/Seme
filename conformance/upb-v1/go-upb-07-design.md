# Go UPB-07 acceptance design

Status: claimed 2026-09-10 by `scripts/check-go-upb-07.sh`. This document
records the bounded acceptance design and explicit target-placement limit.

UPB-07 should extend the complete cumulative UPB-06 project with one neutral,
capability-backed durable-state port. Filesystems, databases, Pulp storage, and
cloud services are realizations of the port; none becomes a Seme Core concept.

## Small bounded proof

The ordinary Go delta should define one durable family with exactly two schema
versions and one migration edge:

```text
example.test/go-uab-11/application-state
    v1 --MigrateV1ToV2--> v2
```

A practical project-neutral data shape is:

```text
StateV1 { State application.State }
StateV2 { Revision i64, State application.State }
```

The migration validates v1, copies its application state, and initializes the
durable revision to one. V2 validation requires a positive durable revision
and delegates existing application-state invariants where applicable. The
bounded update function takes a validated v2 value, increments the durable
revision with checked overflow behavior, and returns either a new v2 value or
a typed error. Validation, migration, and update are pure and do not import a
storage package.

UPB-06 continues to prove `ApplyConfiguredResource` independently. UPB-07 does
not invoke that effect-producing operation between Load and CAS: doing so would
either leak an application effect when CAS conflicts or require a transaction
spanning unrelated authorities. A later contract may explicitly coordinate
those operations; this bounded durable proof does not silently claim it.

Exact error identities must be frozen before implementation. They must
distinguish at least invalid v1, invalid v2, unsupported version, migration
failure, domain update failure, revision overflow, unauthorized storage,
missing key, malformed stored payload, and CAS conflict.

## Neutral Durable State v1 contract

The first contract should describe meaning rather than a database API:

- stable state-family and schema-version identities;
- one current schema and one explicitly declared predecessor;
- a typed, directed migration edge from v1 to v2;
- pure validators for each version;
- canonical payload encoding identity and bounded byte size;
- a stable logical key type;
- a load operation returning `{version, payload, token}` or a typed error;
- compare-and-swap taking the exact load token and one canonical v2 payload;
- explicit `durable-state.read` and `durable-state.compare-and-swap`
  capabilities;
- deterministic operation ordering and typed port results; and
- a Project revision binding the state contract and its provider requirements
  to the cumulative Project-v9 snapshot.

The opaque comparison token belongs to the port realization. Seme code may
thread it but must not inspect, derive, order, or synthesize it. CAS is the only
write operation in this cell; unconditional save, transactions over multiple
keys, queries, leases, deletion, and provider-specific consistency levels stay
outside UPB-07.

The bounded placement is explicit. Pure validation, migration, update, and
planner behavior is claimed for native Go, canonical evaluation, standalone
Wasm, and pinned Pulp. The stateful DurablePort executor is claimed only for
the authenticated Go host boundary. Pinned Pulp must resolve the opaque-token
CAS requirement as unsupported and receives no filesystem, SQLite, or other
ambient storage authority. This does not weaken cross-target agreement: every
target actually claimed for a behavior must agree; DurablePort is simply not a
claimed Pulp behavior until a conforming provider exists.

## Execution route and exact traces

One runtime command follows this fixed route:

```text
authorize read + compare-and-swap
  -> Load(family, key)
  -> decode bounded payload
  -> validate v2 OR validate v1 then migrate v1->v2
  -> pure domain update
  -> validate resulting v2
  -> encode canonical v2 payload
  -> CompareAndSwap(family, key, load-token, payload)
  -> return committed state/result
```

There is at most one Load and at most one CAS attempt. There is never a retry
inside the semantic operation. A caller may issue a new command after a typed
conflict, but that is a new trace.

Required ordered port traces are:

| Case | Trace | Durable commit |
|---|---|---|
| current v2 success | `Load(v2,token)`, `CAS(token,v2')=committed` | exactly v2' |
| v1 migration success | `Load(v1,token)`, `CAS(token,v2')=committed` | exactly migrated-and-updated v2' |
| unsupported/invalid stored payload | `Load(...)` only | none |
| domain validation/update failure | `Load(...)` only | none |
| CAS conflict | `Load(...)`, `CAS(...)=conflict` | none |
| missing create success | `Load(...)=missing(token)`, `CAS(token,v2)=committed` | exactly v2 |
| authorization/input failure | empty trace | none |

“Failure has no state or effects” means no authoritative state transition, no
successful durable write, and no unrelated application effect. A read or
failed CAS already issued to discover a storage-side error remains visible in
the exact port trace; erasing it would be dishonest. Capability denial and
locally malformed inputs reject before the first port operation and therefore
have an empty trace.

## Cumulative corpus

Retain all 2,066 UPB-06 request/observation pairs as the pure application
baseline. Add a small deterministic durable matrix that covers:

- current-v2 success;
- v1 migration plus update success;
- rejected revision zero, accepted revision one, and overflow at maximum;
- invalid v1 and invalid v2;
- unsupported version;
- missing-key create, malformed canonical payload, and missing-load without an
  opaque absence token;
- domain rejection before CAS;
- CAS conflict; and
- missing each required capability.

The observation format must separate the returned typed value, authoritative
state transition, ordered port trace, and other effects. It must include the
exact canonical bytes and digest offered to CAS, without relying on a native
map's iteration order. Replaying the same initial port state and request corpus
must reproduce final durable contents and every observation byte-for-byte.

## Reuse versus new work

Reuse unchanged:

- the UPB-06 ordinary project, resources, 2,066-case corpus, materializer, and
  pure `ApplyConfiguredResource` behavior;
- Execution v36 values, records, results, calls, control flow, bytes, explicit
  transitions, effects, and canonical value ABI where sufficient;
- Package v4 ownership, Dependency v1, Configuration v3, Resource v1, and
  Project v9 evidence;
- native/canonical observation machinery, standalone Wasm cell, pinned Pulp
  runner, deterministic bundle publication, source inventory, projection,
  offline proxy, and atomic-output helpers;
- capability declarations and ordered effect observations from existing
  Foundation/Execution semantics.

New, if proven necessary:

- Durable State v1 semantic contract and validator/emitter;
- a project revision binding it above Project v9;
- a strict Go manifest/adapter for the one family, two schemas, and migration;
- a source-free loader/report and a deterministic in-memory conformance port;
- a runtime-port executor that enforces authorization and one-load/one-CAS;
- Go provider/projector support only for genuinely general expressions exposed
  by the ordinary fixture; and
- target lowering/ABI support only if the exact neutral port route exposes a
  real gap.

No fixture-specific state identity, key, field name, error code, or migration
shape may be recognized by Core, the Go provider, canonical evaluator, Wasm
target, or Pulp runner.

## Seven acceptance evidence classes

1. **Native project.** The cumulative ordinary Go project passes offline. Pure
   v1/v2 validation, migration, and update tests plus deterministic native port
   traces cover the bounded matrix; the prior 2,066 semantics remain unchanged.
2. **Project lift.** One authenticated build owns both schema types, validators,
   migration, update callable, family declaration, capabilities, and port
   requirements. The project revision binds these to the exact Project-v9
   snapshot without flattening package ownership.
3. **Canonical parity.** Canonical pure execution matches native validation,
   migration, update, typed errors, resulting v2 payloads, and absence of
   unrelated effects for every case.
4. **Resolution fidelity.** A source-free loader reproduces the full prior
   authority chain plus Durable State v1, proves the single acyclic v1-to-v2
   migration, exact callable/type ownership, canonical codec, bounds, and
   capability requirements, and emits a deterministic report.
5. **Target and host-boundary placement evidence.** Standalone Wasm and pinned
   Pulp execute the pure durable planner with the same canonical program and
   reproduce its native values. Separately, the authenticated Durable-v1
   artifact derives an immutable host profile that drives the host executor's
   exact family, bounds, capabilities, operation identities/order, and opaque
   token threading against a deterministic in-memory port. This probe proves
   authenticated metadata reaches the host boundary; it does not prove the
   probe transformer implements the declared codec or project domain. The
   pinned Pulp runtime has no matching opaque-token compare-exchange provider,
   so UPB-07 does not claim that Pulp executes DurablePort or produces its CAS
   traces. A future Pulp realization requires separate provider evidence.
6. **Project round trip.** Authenticated projection emits ordinary Go and
   copies opaque project/resource bytes exactly. Native tests pass; exact
   execution meaning and durable family/migration authority re-lift, while
   inventory-bound artifact revisions change honestly and reproduce on a
   second projected lift.
7. **Atomic rejection.** Every adversary below rejects without a claimed
   partial bundle/projection/report, authoritative state transition, successful
   durable write, or unrelated effect. When a Load or failed CAS is required to
   discover the error, only that exact operation remains in the port trace.

## Required adversaries

Contract and family:

- zero, duplicate, unknown, or over-limit family/schema identities;
- no current version, two current versions, missing v1 or v2, and an undeclared
  version;
- missing, reversed, duplicate, branching, cyclic, or extra migration edges;
- migration with wrong parameter/result types, package owner, or source origin;
- validators/update callable with wrong signatures, effects, or ownership;
- codec identity/type mismatch and payload bound zero or over limit;
- missing or extra read/CAS capability requirements;
- Durable State artifact or project-binding tamper and independently valid
  mixed prior/durable/project artifacts.

Runtime and codec:

- unauthorized read, unauthorized CAS, and only-one-capability grants;
- empty, malformed, noncanonical, trailing, wrong-type, and over-limit payload;
- unsupported version and version/payload disagreement;
- invalid v1 before migration, invalid migration output, and invalid v2;
- revision overflow and domain command rejection;
- missing key, duplicate scripted responses, unused scripted operations, and
  out-of-order Load/CAS expectations;
- forged, empty, changed, or reused comparison tokens;
- CAS without Load, two Loads, two CAS attempts, unconditional retry, wrong key
  or family, wrong expected token, and wrong CAS payload;
- CAS conflict that mutates state or emits a success effect;
- provider-realization evidence whose backing store did not commit the exact
  requested bytes after reporting success (the executor cannot infer this from
  the contract response alone);
- request replay from the same initial store producing different trace/state;
- malformed Wasm ABI, denied capability, target trap, and partial output for
  the pure planner target;
- a Pulp placement attempt that treats filesystem or SQLite writes as the
  required opaque-token CAS, or otherwise reports DurablePort supported; and
- any Pulp planner execution receiving ambient storage capability.

Filesystem/database adapter boundaries:

- path traversal, symlink, digest, and create-only adversaries remain covered by
  UPB-06 and must not be redefined as durable-state semantics;
- ambient `os`, database-driver, environment, clock, random, goroutine, or
  global-initializer access in the pure state package rejects;
- provider-specific transaction, SQL, filesystem-locking, or retry claims do
  not count without separate realization evidence.

## Commands

Development evidence is split into fixture, pure semantic, and runtime-port
proofs. Only the final cumulative gate maps the cell:

```text
scripts/check-go-upb-07-fixture.sh       # ordinary/native development evidence
scripts/check-go-upb-07-runtime.sh       # pure planner canonical/Wasm/Pulp evidence
scripts/check-go-upb-07-port-runtime.sh  # host DurablePort boundary evidence
scripts/check-go-upb-07.sh               # mapped seven-class authority gate
```

The mapped final gate passes all seven evidence classes. The profile is 7/36
overall (19.4%), and the Go column is 7/12 (58.3%).
