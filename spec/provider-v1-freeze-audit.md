# Provider Contract v1 Go proof evaluation

Provider Contract v1 is complete for its first explicitly declared Go profile.
This freezes the provider-neutral contract and proves one adapter boundary; it
does not claim general Go support.

Run the evidence with:

```sh
./scripts/check-provider-v1.sh
```

## Gate matrix

| Gate | State | Evidence |
|---|---|---|
| Provider-neutral canonical schema | pass | The 8-schema Provider Contract graph reproduces exactly and passes Kernel and Foundation validation. |
| Ordinary project | pass | The fixture is an ordinary Go module with no Seme dependency or in-project metadata and passes its native tests before import. |
| Scoped profile | pass | Toolchain, language version, target, environment, supported constructs, and exclusions are explicit in every ingestion. |
| Deterministic import | pass | Repeated ingestion without prior evidence produces byte-identical manifest and G1 projection; the compiled envelope passes Kernel and Foundation. |
| Stable opaque identities | pass | Files, declarations, occurrences, evidence, opaque regions, profiles, and ingestion results use opaque 128-bit identities distinct from their matching evidence and revision digests. |
| Canonical semantic edit | pass | The rename is composed into a Foundation-valid Patch v1 workspace and committed by the canonical applicator before projection. |
| Fail-closed reconciliation | pass | An unstamped semantic candidate and a concurrent native edit both reject without changing the project. |
| Minimal projection | pass | Only the definition and type-resolved call occurrence change; an independent oracle reconstructs and compares the exact expected bytes. |
| Native validation | pass | Syntax validation and `go test ./...` run against an isolated staged project before the single changed file is atomically published. |
| Identity recovery | pass | Re-ingestion changes `Greeting` to `Welcome` while preserving its entity identity, its semantic fingerprint, and every unaffected declaration identity and resolved semantic fingerprint. |
| Canonical projection report | pass | Changed file, committed semantic revisions, native command, and status are emitted as a Provider Contract graph passing Kernel and Foundation. |
| Invisibility | pass | `go.mod`, tests, README, and all unrelated bytes remain unchanged; no Seme artifact enters the native project. |

## Frozen canonical authority

| Artifact | SHA-256 |
|---|---|
| Provider Contract v1 module | `7c48adf8097cd6f189569fdfb7db10fdc19f67e573253f63a23e3efa7d6385ea` |
| Checked G1 construction projection | `7e7f3829de10c0010fda4851f53554522816ac6660ccc0704d767d782f6b66fa` |

`module.seme` is canonical authority. The Go implementation and its JSON
manifests are adapter/evidence layers. Canonical Kernel validation, Foundation
validation, and Patch execution do not depend on the Go implementation.

## Exact profile boundary

- module-root package only;
- package-level functions only;
- production Go files are parsed and type checked;
- Go source, tests, `go.mod`, and `go.sum` where present participate in native
  revision/opaque evidence;
- generated files, cgo, methods, build-tag variants, multi-package projection,
  and simultaneous native/canonical edits are excluded;
- one native file may change in one v1 projection transaction.

Broader Go coverage must declare a new supported profile and add conformance
fixtures. A second language provider is the neutrality gate before extracting
a shared provider SDK.
