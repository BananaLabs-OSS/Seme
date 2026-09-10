# Go UPB-05 acceptance design

Status: completed 2026-09-10.

The exploratory design converged on a single same-run Execution-v36 project,
rather than embedding or upgrading earlier v35 project artifacts. Package v4,
Configuration v3, and Project v8 authenticate that complete configured-project
snapshot while older claimed cells and artifacts remain unchanged.

See `conformance/upb-v1/go-upb-05.md` for the final evidence boundary and
`scripts/check-go-upb-05.sh` for its authoritative executable gate.
