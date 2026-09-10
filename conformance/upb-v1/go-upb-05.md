# Go UPB-05 evidence

Status: claimed 2026-09-10 by `scripts/check-go-upb-05.sh`.

The cumulative ordinary Go project retains the complete 2,048-case UAB-11
application and adds ten configuration boundaries. Configuration is explicit
and typed; defaults and validators are pure. Initialization proceeds through
configuration, policy, and service dependencies, producing observable stages
1, 2, and 3 without environment reads, global initialization, clocks, or
random inputs.

One Execution-v36 provider run produces all authoritative evidence. Package v4
binds complete declaration ownership, Configuration v3 binds typed defaults,
validation, runtime input, argument assembly, and the acyclic initializer DAG,
and Project v8 binds the source inventory, dependency closure, package graph,
program, configuration graph, and snapshot revision.

The gate proves seven independent evidence classes:

1. The ordinary five-package Go project and its 2,058 deterministic native
   requests pass offline.
2. Repeated builds reproduce every semantic bundle artifact.
3. The authenticated configuration executor runs three initializers with six
   ordered lifecycle transitions and produces a ready stage-3 runtime.
4. Native, canonical, standalone-Wasm, and pinned-Pulp observations agree for
   all 2,058 requests.
5. The Project-v8 authority report exposes the exact contract and snapshot
   revisions, five owned packages, and three initializers without source access.
6. Authenticated projection produces ordinary Go, preserves opaque module and
   test bytes, passes native tests, and re-lifts to the exact v36 execution
   graph; projected evidence also reproduces independently.
7. Missing or mistyped runtime input, tampered configuration/project artifacts,
   independently valid mixed bundles, ambient configuration, malformed graphs,
   denied capabilities, and existing destinations reject without partial
   output.

The public bundle contains `construction-v36.g1`, `execution-v36.seme`,
`project-base-v8.seme`, `inventory-v8.seme`, `package-detail-v4.seme`,
`package-v4.seme`, `dependency-v1.seme`, `configuration-v3.seme`,
`project-v8.seme`, and `COMPLETE.sha256`.

This bounded claim does not imply general Go configuration frameworks,
arbitrary package initialization, ambient environment compatibility, or broad
Go project support.
