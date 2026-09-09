# Go UAB-09 evidence

Status: complete. `scripts/check-go-uab-09.sh` binds all five evidence classes
to the scorecard. Idiomatic `log.Print` calls lift to explicit
`observability.log` effect invocations protected by the matching capability.
Original/projected Go, canonical evaluation, standalone Wasm, and pinned Pulp
agree on return values and exact ordered traces for every Boolean pair. Denied
execution emits no partial trace. Malformed requests, unsupported source
effects, and forged effect-to-capability links reject.
