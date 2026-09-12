# Shared UPB-12 acceptance design

Status: claimed by `scripts/check-upb12-shared.sh`.

UPB-12 is one cross-language claim, not three unrelated demonstrations. One
canonical project authority is the source of meaning. Go, JavaScript, and Lua
are independent native projections of that authority and receive no privileged
language-specific semantic branch.

## Cumulative project

The fixture is a deterministic command service with three logical packages:

- `domain` owns commands, explicit state/result transitions, typed
  configuration, injected clock and seeded-random inputs, and replay;
- `durable` exposes the versioned compare-exchange state port and migration;
- `transport` exposes ordered framed command/reply delivery and one
  capability-authorized external effect.

The project also carries one digest-addressed UTF-8 resource. Its closure,
configuration order, resource identity, port requirements, capabilities,
target placement, revision history, and reconciliation patch are represented
by the existing neutral contracts. Language ecosystems own only native package,
source, toolchain, and runtime realization mechanics.

## Projection authority

The canonical project is constructed once without native source provenance.
Each projector independently emits an ordinary ecosystem project into a new
destination:

- Go packages and an offline `go test ./...` entry point;
- JavaScript ES modules and an offline Node test entry point;
- Lua modules and an offline Lua test entry point.

Removing Seme evidence leaves each projection runnable and testable. Re-lifting
each complete projection must reproduce the exact authoritative semantic graph
bytes. Native spelling, module syntax, integer representation, error idioms,
and runtime adapters may differ only through authenticated presentation or
realization evidence; those differences cannot enter neutral Core meaning.

## Shared evidence

One cumulative gate must:

1. authenticate the source-free canonical project and independently project,
   build, test, and re-lift all three native projects;
2. compare at least 4,096 generated command observations across all native
   projections, canonical evaluation, supported standalone Wasm, and pinned
   Pulp realization;
3. prove exact dependency, configuration, resource, state, transport, effect,
   time, randomness, replay, placement, and revision authority;
4. apply the same identity-bound semantic patch through each native projection,
   validate it natively, and recover the same resulting canonical revision;
5. reproduce the entire publication twice byte-for-byte; and
6. reject dependency, resource, revision, capability, boundary-value, graph,
   mixed-bundle, symlink, native-test, and destination-collision adversaries
   without partial publication.

No language column may claim UPB-12 before this shared gate passes. Earlier
JavaScript and Lua UPB cells may reuse Go-proven neutral contracts, but must
supply their own native project, lift, projection, parity, and rejection
evidence. A mixed-language deployment is additional evidence only.

## Implementation order

1. Extract a language-neutral cumulative fixture descriptor and expected
   Project-v14 authority from the proven Go fixture.
2. Add strict JavaScript project discovery, package closure, projection, and
   native validation, then advance its UPB cells cumulatively.
3. Add the equivalent strict Lua project path and advance its UPB cells.
4. Build a source-free shared projector/re-lift gate and certify UPB-12 for all
   three languages simultaneously.
5. Exercise the resulting language service and project authority from a bounded
   external Workbench consumer without adding Workbench concepts to Seme.
