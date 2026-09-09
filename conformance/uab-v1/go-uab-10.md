# Go UAB-10 evidence

`scripts/check-go-uab-10.sh` is the complete evidence gate for Go UAB-10.
It proves stable semantic declaration identity across an ordinary `gofmt`
revision, a revision-bound canonical rename, minimal native projection, and
prior-evidence re-import.

- **Lift:** the real Go parser and type checker ingest the package before and
  after formatting and emit validated Provider Contract v1 graphs.
- **Native parity:** native tests pass before formatting, after formatting,
  and after the projected rename; the rename updates its resolved call site.
- **Target parity:** the cumulative deterministic Wasm and pinned-Pulp gate
  remains green for the same typed function/call semantic family.
- **Projection round trip:** every declaration and evidence identity survives
  formatting, while the renamed declaration retains its identity and resolved
  semantic fingerprint after minimal projection and re-import.
- **Rejection:** stale revisions and unstamped candidates cannot touch source;
  ambiguous prior identity evidence is rejected with a source location.

The identity manifest is reconciliation evidence, not application metadata.
No downstream application or editor concept is embedded in Seme.
