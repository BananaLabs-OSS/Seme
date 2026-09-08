# Seme progress dashboard

Snapshot: 2026-09-08 after Core v24.

Percentages answer different questions and must not be conflated:

| Goal | Progress | Basis |
|---|---:|---|
| Defined Core restoration checkpoints | 79% | 23 of the 29 additive checkpoints from v2 through v30 have been reached; many remain intentionally marked in progress until broader composition evidence exists. |
| Collection sequence | 80% | v21 through v24 of the five-checkpoint v21–v25 sequence are implemented. |
| Bounded Go/JavaScript semantic bridge | about 48% | Estimate for the deliberately supported, tested language subset—not either complete language. |
| Useful small-application subset | about 38% | Estimate against the semantics needed for modest deterministic applications. |
| Broad Go/JavaScript ecosystem compatibility | under 5% | Real packages depend on much larger language, runtime, standard-library, build, and foreign-interface surfaces. |
| Universal-language vision | about 2% | Only Go and JavaScript have bounded adapters; universal multi-runtime interoperability remains a long-horizon goal. |

The checkpoint percentage measures delivery against a finite internal roadmap.
It is not a claim that Seme supports the same percentage of Go, JavaScript, or
all programming languages. Practical percentages should move only when real
programs, runtime contracts, and adversarial acceptance gates justify them.
