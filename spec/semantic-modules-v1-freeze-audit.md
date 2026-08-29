# Semantic Foundation v1 and Patch Module v1 freeze evaluation

Semantic Foundation v1 and Patch Module v1 are complete at their stated v1
boundaries. “100%” here means every normative behavior in
`semantic-foundation-v1.md` and `patch-module-v1.md` has canonical executable
authority and reproducible conformance evidence. It does not claim that future
mechanics, provider contracts, package resolution, or arbitrary constraint
languages are already implemented.

Run the evidence with:

```sh
./scripts/check-kernel-v1.sh
./scripts/check-sha256-v1.sh
./scripts/check-semantic-modules.sh
```

## Foundation v1 gate matrix

| Gate | State | Evidence |
|---|---|---|
| Canonical declarations | pass | The generated G1 projection exactly reconstructs the authoritative 61-entity Foundation graph. |
| Cardinality and recursive shapes | pass | The canonical validator executes required, optional, many, list, record, reference, primitive, hole, and any-shape checks. |
| Versions and imports | pass | Active fields validate by schema version; future versions remain preserved unknown; unresolved exact-revision imports remain unavailable. |
| Preservation | pass | Unknown fields, unknown versions, unavailable schemas, and opaque values survive without false certification. |
| Rejection | pass | Malformed known declarations and values reject with status 65 and a canonical stable parent diagnostic. |
| Successful report | pass | The canonical reporter emits one ordered disposition per input entity and an independent Go decoder verifies totals. |
| Canonical execution | pass | Validator and reporter lower from their canonical graphs to byte-identical K0 execution artifacts. |

## Patch v1 gate matrix

| Gate | State | Evidence |
|---|---|---|
| Provider-neutral schema | pass | Patch and RenameDeclaration use stable identities and bytes-addressed target/field identities rather than host pointers or a Go-specific AST. |
| Base and preconditions | pass | Base revision equality and every expected value are checked against the unchanged workspace. |
| Resolution and conflicts | pass | Unknown targets, unknown fields, and duplicate writes are independently reachable and deterministic. |
| Isolated candidate | pass | Valid operations construct a separate zero-revision candidate exactly equal to the independent applied fixture. |
| Candidate validity | pass | Empty declaration names reach candidate validation and deterministically reject as `patch.invalid_candidate`. |
| Revision authority | pass | The transcript is domain-separated, hashed by canonical portable SHA-256, and stamped with its first 128 bits; the Go oracle independently recomputes it. |
| Atomic commit | pass | Only a Kernel- and Foundation-valid stamped candidate is renamed into the requested output. Every rejection leaves a pre-existing output byte-for-byte unchanged. |
| Required diagnostics | pass | All six stable rules emit canonical reports with revision, operation index, available target/field, and expected/actual bytes. Kernel, Foundation, and an independent Go decoder verify every report. |
| Canonical execution | pass | Applicator, diagnostic emitter, transcript builder, and revision stamper lower from canonical graphs to byte-identical K0 artifacts. |

## Frozen canonical authorities

| Artifact | SHA-256 |
|---|---|
| Foundation module | `bbb42f8d71f8c537713a478514f74f79cef60ba370b1a2e10525f631d922dd5d` |
| Foundation validator | `0a2ab3d03c4ee0c212352ae4e340ed9b72c2ee1f78f1610ebf7bde24c5eda168` |
| Foundation reporter | `d137029e3a43532f15630d5853b704a3b5e71d6db80db3dabb6ca691ecc97429` |
| Patch module | `f7ec1943de753c143d6a36b87270a7c898262b167ed45745ada7d3099292ce12` |
| Patch candidate applicator | `5a6802ac56906371a8091cf7dd5cd65841ada7c335dc609c665b348ce8bd00a9` |
| Patch diagnostic emitter | `018fa00053edf93319bb09dfd470add1fdb915530760a0ee89d38f8929649993` |
| Patch revision transcript | `d64a7ca622d8a1c49650eae7aa5dbfb6312c7c9ee9bccf0dc71b3d5db87a6c4a` |
| Patch revision stamper | `e174c91c21b6857c390b804f909d8c2c487306e612731f457b240ddb3b26bf89` |

The `.seme` graphs are authoritative. S1 is retained construction history, G1
is a checked readable projection, K0 is derived executable evidence, and Go is
only an independent fixture/oracle layer.

## Deliberately after this freeze

- Provider Contract v1 and stable source reconciliation.
- The first lossless Go import/edit/project/re-import proof.
- General constraint-operation languages beyond Foundation v1 shape and
  cardinality validation.
- Package resolution, effects/capability policy, mechanics composition, and
  target lowering beyond the contracts represented by these modules.

Those are consumers or later semantic modules. They do not reopen Foundation
v1 or Patch v1 unless a frozen guarantee above is found false.
