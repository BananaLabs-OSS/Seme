# Go UPB-04 evidence

The mapped acceptance gate is `scripts/check-go-upb-04.sh`.

The gate exercises the seven evidence classes over an ordinary materialized
copy of the full three-package Go UAB-11 application and all 2,048 cumulative
command observations:

1. The untouched project builds and passes its native Go tests using only the
   pinned repository-local dependency proxy.
2. One production import emits and validates Project v5: Project v4 preserves
   the resolved dependency closure, while Package v3 owns every canonical
   record, interface, receiver method, generic realization, function, and
   declared effect without flattening package identity.
3. Canonical execution agrees with the native behavioral oracle for all 2,048
   stateful commands, including values, typed failures, state transitions, and
   ordered effect traces.
4. A deterministic Project v5 report exposes the exact Package v3 ownership,
   local import, effect requirement, and pinned metadata-only ecosystem
   dependency. The latter is not called and does not imply ecosystem-package
   interoperability.
5. Standalone Wasm and the pinned Pulp host execute the same canonical graph,
   recursive ABI requests, and authorized effect boundary with identical
   observations.
6. Projection consumes authenticated Package v3 ownership, produces three
   ordinary Go packages, builds natively, and re-lifts to byte-identical
   canonical meaning. Repeated import and projection are deterministic; opaque
   module, checksum, and test files retain their contracted bytes.
7. Forged ownership, undeclared or private calls, missing package/effect
   requirements, generic-family ownership tampering, dependency tampering,
   malformed graph/ABI input, capability denial, symlink traversal, and an
   existing output destination reject without a partial report, projection,
   bundle, state transition, or effect trace.

The gate treats a generic realization's type arguments as instantiation edges,
not reverse package dependencies. A package that declares `Result` does not
acquire a dependency on every package whose types instantiate it; the use site
already records the directional package import.
