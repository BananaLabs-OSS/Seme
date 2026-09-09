# Seme progress dashboard

Snapshot: 2026-09-09 after the Core v30 cumulative composition proof.

Percentages answer different questions and must not be conflated:

| Goal | Progress | Basis |
|---|---:|---|
| Defined Core restoration checkpoints | 100% | All 29 additive checkpoints from v2 through v30 have been reached. |
| Core architectural composition gate | 100% | Both frozen cumulative proofs compose existing semantics across Go, JavaScript, canonical Seme, deterministic Wasm, and pinned Pulp without a program-specific provider or target recognizer. |
| Collection sequence | 100% | All five planned checkpoints v21 through v25 are implemented and independently gated. This is completion of that bounded sequence, not all collection behavior. |
| Bounded Go/JavaScript semantic bridge | about 82% | Estimate for the deliberately supported, tested language subset—not either complete language. |
| Useful small-application subset | about 75% | Estimate against the semantics needed for modest deterministic applications. |
| Useful Application Bridge Profile v1 | 6/36 cells (16.67%) | UAB-01 and UAB-02 pass all five evidence classes independently for Go, JavaScript, and Lua. The other 30 cells remain unclaimed. |
| UAB-02 exact value bridge | complete | All three languages pass shared scalar, aggregate-collection, and tagged-composite observations through original and projected native source, structural canonical evaluation, standalone Wasm, and pinned Pulp, with representation-appropriate rejection evidence. |
| Broad Go/JavaScript ecosystem compatibility | under 5% | Real packages depend on much larger language, runtime, standard-library, build, and foreign-interface surfaces. |
| Universal-language vision | about 2% | Only Go and JavaScript have bounded adapters; universal multi-runtime interoperability remains a long-horizon goal. |

The checkpoint percentage measures delivery against a finite internal roadmap.
It is not a claim that Seme supports the same percentage of Go, JavaScript, or
all programming languages. Practical percentages should move only when real
programs, runtime contracts, and adversarial acceptance gates justify them.
