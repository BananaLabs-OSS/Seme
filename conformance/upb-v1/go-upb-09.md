# Go UPB-09 evidence

Status: claimed 2026-09-11 by `scripts/check-go-upb-09.sh`.

The cumulative ordinary Go project retains UPB-08 and adds one controlled
command transition with explicit clock samples, a versioned seeded random
transition, one ordered external Boolean effect request, and deterministic
replay. Controlled Effects v1 describes the neutral contract. Project v12
binds its exact plan to the complete Project-v11 snapshot.

The gate proves all seven mapped evidence classes:

1. The cumulative ordinary project passes race tests and a deterministic
   4,096-observation native controlled-effects corpus, including clock bounds,
   seed extrema and overflow, duplicates, gaps, conflicts, and domain rejects.
2. Repeated real-project lifts reproduce the authenticated Project-v12 graph,
   effects plan, replay authority, and complete closed-world bundle.
3. Native and canonical evaluation agree for every complete state, result,
   random transition, clock observation, and logical effect trace.
4. Source-free loading and reporting reproduce the complete predecessor,
   selected providers, bounds, identities, replay, resources, and package
   authority without consulting original source.
5. Standalone Wasm and pinned Pulp agree with all 4,096 pure observations. Live
   clock sampling and effect delivery remain an authenticated Go host boundary;
   Pulp's observed parity harness records the logical request only, and ambient
   WASI clock or entropy providers reject as substitutions.
6. Authenticated projection emits a self-contained ordinary Go module,
   reconstructs pinned dependency metadata, builds offline, re-lifts to exact
   canonical construction and Execution bytes, and retains honest differences
   in source-bound Project-v12 provenance.
7. Invalid seeds/clocks, inconsistent random or replay ledgers, forged effect
   identities/payloads, denied capabilities, ambient substitutions, mixed or
   tampered artifacts, unsafe projection, and existing destinations reject
   without partial authority, graph, source, state, or effect commits.

This bounded claim does not imply ambient time, host entropy, cryptographic
randomness, arbitrary effects, exactly-once external delivery, general Go
runtime compatibility, or arbitrary Go project import.
