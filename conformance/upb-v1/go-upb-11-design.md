# Go UPB-11 acceptance design

Status: claimed 2026-09-11 by `scripts/check-go-upb-11.sh`.

Go UPB-11 extends the cumulative Project-v13 fixture with live, transactional
project revision. It composes the existing neutral Patch v1 and Live Language
Service v1 contracts; it does not add Go files, editor behavior, filesystem
mechanics, Workbench, or merge algorithms to Core semantics.

## Neutral Project-v14 authority

Project Contract v14 adds one `ReconciledProjectRevision` record. An accepted
record binds:

- SHA-256 bindings to the exact prior and resulting Project-v13
  `PlacedProjectSnapshot` artifacts;
- one exact Patch-v1 transaction containing the identity-bound semantic edit;
- a strictly increasing project client revision;
- the prior and resulting 32-byte project content revisions; and
- a digest of the independently observed native validation transcript.

The record describes an accepted project revision; it does not make a patch,
source projection, native test, re-lift, or target plan valid. Rejected and
incomplete editor snapshots remain Live Language Service results and never
become Project-v14 authority. Project v14 imports exact Project v13, Patch v1,
and Live Language Service v1 revisions.

## Revision coordinator

The Go provider receives complete in-memory multi-file snapshots. One session
owns a strictly increasing project client revision, its current accepted source
snapshot, last-valid canonical graph and mappings, exact Project-v13 authority,
and immutable source digests. Arrival order has no authority: only the declared
revision and base identities do.

An invalid but newer snapshot advances the observed client revision while
retaining the complete last-valid project. A stale snapshot changes nothing.
No partially lifted package, projected source, plan, or deployment becomes
observable.

## Identity-bound semantic edit

The bounded edit renames one exported declaration owned by a non-root package
and every typed reference to it from another package. The request names the
canonical declaration identity, its field identity, expected value, replacement
value, Patch base revision, project base content revision, and client revision.
Source locations are evidence used to project the canonical decision; they are
not identity authority.

The coordinator must:

1. authenticate the complete Project-v13 base and detached source bundle;
2. reject a stale client, patch, project, plan, catalog, or launch revision;
3. apply Patch v1 to an isolated canonical candidate;
4. resolve every declaration and reference occurrence from typed provider
   mappings derived from the unchanged base snapshot;
5. reject ambiguity, overlapping writes, or a source and semantic target that
   both changed from their common base;
6. project all affected files into an isolated directory while preserving
   every untouched and opaque byte contract;
7. run the declared native formatter/build/test commands without network
   resolution;
8. re-import the complete projected project and require the edited canonical
   identities, types, package ownership, calls, effects, and observations;
9. regenerate target requirements, provider catalog, plan, Project-v13,
   artifacts, and launch authority from the new project; and
10. atomically publish source plus Project-v14 evidence only after every check
    succeeds.

This is semantic reconciliation, not blind textual replacement. Comments and
unrelated identifiers matching the spelling remain unchanged. Go-specific
identifier and formatting rules belong to the Go projector.

## Conflict policy

V1 supports a single common-base, one-sided semantic edit. These conditions
reject rather than being guessed:

- the client revision is zero or not greater than the latest observed one;
- the patch base or project content revision is stale;
- the named semantic identity or field is absent or no longer has the expected
  value;
- source bytes covering a mapped declaration/reference changed independently;
- both the canonical semantic value and corresponding native source changed;
- multiple declarations or references claim the same mapped range;
- affected ranges overlap or cross file bounds;
- the replacement is invalid in Go, changes visibility unexpectedly, collides
  in a package scope, or produces a native/canonical mismatch; or
- any untouched, generated, vendored, ignored, opaque, resource, dependency,
  configuration, runtime-boundary, target, or deployment authority drifts.

There is no automatic three-way merge in this cell. A later provider may offer
one, but it must report its own evidence and cannot relabel a conflict as exact.

## Determinism and evidence

Equal base authority, source bundle, patch, revision, toolchain pins, target
catalog, and validation commands produce byte-identical projected sources,
Project-v14 graph, plan, deployment bundle, reconciliation report, and native
validation transcript digest. A successful rename changes the appropriate
source and project revisions while stable semantic identities remain stable.

The cumulative gate must prove all seven UPB evidence classes:

1. the complete ordinary Go project passes its native race tests and inherited
   4,096-observation corpus before and after the edit;
2. complete snapshots at valid, invalid, recovered, and stale revisions have
   deterministic lift and last-valid behavior;
3. native and canonical observations remain equal after the cross-package edit;
4. dependencies and all four native runtime boundaries remain authenticated
   with unchanged fidelity unless their meaning actually changes;
5. standalone Wasm and pinned Pulp retain the inherited observation parity;
6. the atomic projected project builds offline and re-lifts to the edited
   canonical fixed point with untouched/opaque byte preservation; and
7. stale, both-changed, ambiguous, conflicting, malformed, tampered,
   over-limit, native-test-failing, and output-collision adversaries expose no
   partial source, graph, state, effect, plan, artifact, or launch commit.

## Claim rule

Only `scripts/check-go-upb-11.sh` may claim or map Go UPB-11. It must execute
the complete mapped UPB-10 gate, reproduce Project-v14 independently, exercise
the live revision and reconciliation partitions, cover all seven evidence
classes, and perform a clean deterministic second edit/build. The gate passed;
the scorecard is 11/36 overall and 11/12 for Go.
