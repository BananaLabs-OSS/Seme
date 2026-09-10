# Durable State v1 runtime boundary evidence

This package is the project-neutral host-side enforcement layer for an
authenticated Durable State v1 plan. `AuthenticatedProfile` accepts only an
exactly validated `durableinstance.Inputs`; a private fingerprint prevents a
caller from changing identities, ordering, codec, family, current version, or
bounds afterward.

`Execute` preflights both distinct capabilities and the logical key before
I/O, performs exactly one Load and at most one compare-exchange, validates
canonical payload size and SHA-256, never gives a comparison token to domain
code, and threads the exact token into compare-exchange. It has no retry path.
A saved response is accepted only when the provider reports the exact offered
version, bytes, and digest. The tests use a deterministic in-memory adapter to
prove exact final storage and traces for missing-create, migration, update,
validation and transformation failures, authorization failures, malformed
provider outcomes, load/CAS failure, and conflict.

`ExecuteAuthenticated` is the direct path from validated Durable-v1 instance
metadata to that host executor. The source-free UPB-07 report exercises it
with an in-memory profile probe and records the authenticated family and exact
Load/CAS identities and order. That probe establishes metadata-to-host-boundary
placement and opaque-token threading only; its small transformer is not
evidence for the declared codec or the project's domain semantics.

This is host-boundary conformance evidence, not Core or Execution semantics.
It cannot make a dishonest provider atomic; provider realization evidence must
prove that a reported save actually committed the exact bytes and that every
failure/conflict left storage unchanged.

The currently pinned Pulp runtime exposes scoped filesystem and SQLite
capabilities, but no versioned opaque-token compare-exchange primitive matching
this contract. Therefore `storage.fs` or `storage.sqlite` is not claimed as a
DurablePort realization. A future Pulp adapter must implement and separately
prove this exact boundary rather than translating it to an unconditional write
or hidden retry.
