# Go UPB-06 acceptance design

Status: implementation evidence in progress. This document makes no completion
claim and does not change the scorecard.

UPB-06 is cumulative over the complete UPB-05 project. Its new boundary is a
strict project-owned resource declaration resolved into detached,
content-addressed blobs, a Resource-v1 manifest, and a Project-v9 snapshot.
Resource acquisition is adapter policy; filesystem behavior is not Seme Core.

## Fixed ordinary project evidence

`scripts/materialize-go-upb06-fixture.sh NEW_DIRECTORY` produces the cumulative
ordinary Go project. It contains:

- `resources/notice.txt`: `Seme resources — 世界` plus one newline, exactly 26
  bytes, SHA-256
  `dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48`;
- `resources/marker.bin`: `00 ff 53 45 4d 45 0a`, exactly seven bytes, SHA-256
  `d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d`;
- a strict `resources.json` declaration;
- pure typed `resource.Set` validation; and
- `service.ApplyConfiguredResource`, which rejects invalid resource values
  before delegating to the UPB-05 entry.

The deterministic corpus has 2,066 request/expected pairs: the prior 2,048
UAB11 cases and ten configuration cases retain their behavior under the valid
resource set, followed by eight bounded valid/invalid resource cases.

## Exact bundle surface

The authenticated build must publish, create-only, exactly:

```text
construction-v36.g1
execution-v36.seme
project-base-v8.seme
inventory-v8.seme
package-detail-v4.seme
package-v4.seme
dependency-v1.seme
configuration-v3.seme
project-v8.seme
resource-v1.seme
project-v9.seme
blobs/dde7c6f27c3a259a48b5c9e6b886a7f3c1c75f87c73699536eccb7149d1e7d48
blobs/d57b9b19e900f27112610f80812bf55d383d69ffd0e5796b7e1c84164ef15f9d
COMPLETE.sha256
```

`COMPLETE.sha256` is written last and authenticates every canonical artifact
and detached blob. Two identical offline builds must reproduce every byte.

## Seven evidence classes

1. **Native project.** All ordinary Go tests pass offline. Two independently
   emitted 2,066-case corpora are byte-identical. The declared resource files
   are regular files with exact byte lengths and digests.
2. **Project lift.** The unchanged `ApplyConfiguredResource` entry lifts under
   Execution v36. Package v4 retains the typed `resource.Set` ownership and
   BytesEqual realization. Resource v1 binds both declared resources to exact
   authenticated source units and one package owner. Project v9 binds the
   complete Project-v8 snapshot and Resource-v1 manifest.
3. **Canonical parity.** Canonical evaluation matches all 2,066 native
   observations, including typed errors 40 and 41 and absence of effects on
   rejection.
4. **Resolution fidelity.** A source-free loader revalidates and reproduces the
   complete Project-v8, Resource-v1, and Project-v9 chain, verifies every
   detached blob, rejects orphan blobs, and emits a deterministic authority
   report containing both resource identities, destinations, sizes, digests,
   and the Project-v9 snapshot revision.
5. **Target parity.** Standalone Wasm and the pinned Pulp commit consume the
   same Execution-v36 graph and encoded requests and match all 2,066 native
   observations.
6. **Project round trip.** Authenticated projection copies the declared
   resource bytes and opaque project-owned files exactly, emits ordinary Go,
   passes native tests, and re-lifts to the exact Execution-v36 behavior and
   the same resource identities, destinations, bytes, sizes, and digests.
   Inventory-bound Resource-v1 and Project-v9 bytes correctly change when
   projected Go source bytes replace the originals; two re-lifts must reproduce
   those new authority bytes exactly. Projection is create-only and reproducible.
7. **Atomic rejection.** Every malformed or mismatched input below fails with
   no claimed output, no partial destination, and no modification of an
   existing destination.

## Required adversaries

- declaration traversal and absolute/backslash path forms;
- missing declared source;
- source byte/digest/size mismatch;
- duplicate identities and colliding destinations;
- per-resource, total-byte, and resource-count limits;
- source or destination symlinks and symlinked parents;
- malformed/unknown declaration fields and trailing JSON;
- missing, tampered, renamed, extra, or orphan detached blobs;
- tampered Resource-v1 or Project-v9 artifacts;
- Resource-v1 from one valid build mixed with Project-v9 or Project-v8 from
  another valid build;
- denied effects, malformed Execution graph, and existing output directory.

The committed nearby adversary declarations live in
`fixtures/go-upb06-resource-overlay/adversaries/`. A gate must create symlinks
only inside its temporary directory.

## Commands

Foundation-only native evidence:

```sh
scripts/check-go-upb-06-fixture.sh
```

The already implemented source-to-runtime proof (not an authority claim):

```sh
scripts/check-go-upb-06-runtime.sh
```

The final claimed command will be:

```sh
scripts/check-go-upb-06.sh
```

It must invoke the strict `go-upb06-build` and source-free UPB-06 loader/project
CLI, perform authenticated projection and exact re-lift, execute all
adversaries, and only then be added to the claimed-gates runner.
