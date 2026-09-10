# Go UPB-06 evidence

Status: claimed 2026-09-10 by `scripts/check-go-upb-06.sh`.

The cumulative ordinary Go project retains the 2,058 UPB-05 observations and
adds eight bounded resource cases. It owns a 26-byte UTF-8 notice containing
non-ASCII text and a seven-byte binary marker containing `00` and `ff`.
`resource.Set` validation is pure; acquisition remains adapter policy and adds
no filesystem behavior to Seme Core.

Resource v1 authenticates each logical identity, Package-v4 owner,
source-inventory unit, relative path, media kind, size, SHA-256 digest, and
collision-free destination. Project v9 binds the Resource-v1 manifest to the
complete configured Project-v8 snapshot. Content remains detached in
digest-addressed blobs and is independently verified.

The gate proves all seven mapped evidence classes:

1. The ordinary project and two independently generated 2,066-case native
   corpora pass offline and reproduce byte-for-byte.
2. Repeated builds reproduce all semantic artifacts, both blobs, and the
   completion manifest.
3. Native and canonical observations agree for all 2,066 requests.
4. A source-free loader reproduces Project v8, Resource v1, and Project v9,
   verifies the exact detached store, and emits a deterministic report.
5. Standalone Wasm and pinned Pulp match every native observation.
6. Authenticated projection emits ordinary Go, copies resource bytes exactly,
   passes native tests, re-lifts to exact Execution-v36 meaning and identical
   resource authority, and reproduces its new inventory-bound artifacts.
7. Path, source, digest, destination, limit, field, symlink, blob, artifact,
   bundle-shape, changed-byte, and existing-output adversaries reject without
   partial output.

This bounded claim does not imply arbitrary embedding, transforms, streaming,
asset pipelines, platform packaging, or general Go ecosystem support.
