# Go UPB-02 evidence

The mapped gate is `scripts/check-go-upb-02.sh`.

It proves the bounded three-package fixture through seven independent evidence
classes: native project execution; authenticated Project v3 lifting; canonical
behavior parity; exact package, member, visibility, import, alias, ownership,
span, and digest fidelity; standalone Wasm and pinned Pulp target parity;
strict Package v2 native projection and deterministic re-lift; and atomic
rejection of missing imports, cycles, cross-package private access, tampering,
and existing destinations.

Project v1 is byte-identical across native projection because it records
semantic behavior. Project v3 additionally commits source paths and digests,
so it is byte-identical across revisions of the same source tree rather than
across intentionally rewritten source presentations.
