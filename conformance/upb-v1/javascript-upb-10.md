# JavaScript UPB-10 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-10.sh`.

The cumulative gate passed all seven mapped evidence classes. Native
JavaScript, canonical Seme, standalone Wasm, and pinned Pulp retain the full
UPB-09 command/state/replay/effect observations. The authenticated Project-v12
graph deterministically derives 35 requirements: 31 exact realizations and four
explicit native islands.

The four islands are live clock sampling, external Boolean effect delivery,
DurablePort, and TransportPort. No requirement is silently adapted, emulated,
embedded, or dropped. Exact-only policy reports those four requirements as
impossible with four diagnostics, produces zero boundaries, and publishes no
Project-v13 deployment.

Two independent mixed-policy placements are byte-identical. Their provider
catalog, Target-v1 plan, Project-v13 graph, canonical VM, pinned Pulp cell,
native-island launch manifest, source-free report, and completion manifest
reopen under strict authentication. Modified plans and colliding destinations
reject without partial output, and both semantic envelopes pass closed-world
validation.

The run used the already-cached Go 1.26.0 toolchain with process-local
`GOROOT`, `PATH`, and `GOTOOLCHAIN=local` because the workstation's default Go
launcher is 1.25.6 and network toolchain verification is unavailable in the
sandbox. No toolchain was installed and no persistent build setting changed.
