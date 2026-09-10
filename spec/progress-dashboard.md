# Seme progress dashboard

Snapshot: 2026-09-10 after the bounded Go UPB-05 configured-project proof.

Percentages answer different questions and must not be conflated:

| Goal | Progress | Basis |
|---|---:|---|
| Defined Core restoration checkpoints | 100% | All 29 additive checkpoints from v2 through v30 have been reached. |
| Core architectural composition gate | 100% | Both frozen cumulative proofs compose existing semantics across Go, JavaScript, canonical Seme, deterministic Wasm, and pinned Pulp without a program-specific provider or target recognizer. |
| Collection sequence | 100% | All five planned checkpoints v21 through v25 are implemented and independently gated. This is completion of that bounded sequence, not all collection behavior. |
| Bounded Go/JavaScript semantic bridge | about 82% | Estimate for the deliberately supported, tested language subset—not either complete language. |
| Useful small-application subset | about 75% | Estimate against the semantics needed for modest deterministic applications. |
| Useful Application Bridge Profile v1 | 36/36 cells (100%) | UAB-01 through UAB-11 pass all five evidence classes independently for Go, JavaScript, and Lua. UAB-12 projects one shared application into all three languages, requires byte-identical re-lift, and passes 2,048 observations through native, canonical, Wasm, and pinned-Pulp realizations. This is 100% of the frozen bounded profile, not complete language support. |
| Useful Project Bridge Profile v1 | 5/36 cells (13.9%); Go 5/12 (41.7%) | Go UPB-01 through UPB-05 each pass all seven mapped evidence classes. JavaScript and Lua project cells remain unclaimed. |
| Go UPB-02 package graph | complete for the bounded profile | Package v2 and Project v3 commit full membership, native visibility, import bindings, aliases, source ownership, spans, and digests. The production three-package pipeline, strict projection, authenticated report, native/canonical/Wasm/Pulp parity, deterministic re-lift, and fail-closed adversaries pass. |
| Go UPB-03 dependency closure | complete for the bounded profile | Dependency v1 and Project v4 commit a pinned local-plus-ecosystem closure with integrity and source evidence. Go applicability binds it to the exact package edge, source inventory, and `go.mod`; offline repeat, projection, native/canonical/Wasm/Pulp parity, and fail-closed adversaries pass. External dependency calls remain outside this cell. |
| Go UPB-04 typed package behavior | complete for the bounded profile | Package v3 and Project v5 authenticate complete declaration ownership and typed local cross-package calls. The three-package cumulative application passes 2,048 canonical/Wasm/Pulp observations, native projection, exact semantic re-lift, deterministic reporting, and fail-closed adversaries. Ecosystem calls remain unclaimed. |
| Go UPB-05 configuration and initialization | complete for the bounded profile | Execution v36, Package v4, Configuration v3, and Project v8 authenticate one five-package configured project. Three ordered initializers and 2,058 native/canonical/Wasm/Pulp observations agree; ordinary-Go projection re-lifts exactly and adversaries fail atomically. Ambient configuration remains unclaimed. |
| UAB-02 exact value bridge | complete | All three languages pass shared scalar, aggregate-collection, and tagged-composite observations through original and projected native source, structural canonical evaluation, standalone Wasm, and pinned Pulp, with representation-appropriate rejection evidence. |
| UAB-03 compositional control flow | complete | All three languages pass shared locals, mutation, modular arithmetic, comparison, hazardous short-circuit, conditional, loop, and early-return observations through all five evidence paths. |
| UAB-04 collection operations | Go/JavaScript/Lua complete | All three languages pass construction, length, checked indexing, traversal, immutable update/append/removal, and map lookup/insert/removal through original/projected native code, canonical evaluation, standalone Wasm, and pinned Pulp with direct rejection evidence. |
| UAB-05 bounded dynamic dispatch | Go/JavaScript/Lua complete | All three bridges pass frozen structural interfaces/protocols, exact witnesses, native equivalents, and bounded dynamic dispatch through all five evidence paths; language-specific object-model behavior remains outside the exact neutral subset. |
| UAB-06 through UAB-10 | Go/JavaScript/Lua complete | The bounded profile proves closures, explicit transitions, fallibility, authorized effects, and stable semantic identity independently in all three languages. |
| UAB-11 cumulative application | Go/JavaScript/Lua complete | Each bridge independently passes the 2,048-command cumulative stateful application through original/projected native code, canonical execution, standalone Wasm, pinned Pulp, and adversarial rejection. |
| UAB-12 exact cross-language application | complete | One canonical application projects to Go, JavaScript, and Lua; all projections re-lift byte-identically and produce the same 2,048 observations across every realization. |
| Broad Go/JavaScript ecosystem compatibility | under 5% | Real packages depend on much larger language, runtime, standard-library, build, and foreign-interface surfaces. |
| Universal-language vision | about 2% | Go, JavaScript, and Lua have bounded adapters; universal multi-runtime interoperability remains a long-horizon goal. |

The checkpoint percentage measures delivery against a finite internal roadmap.
It is not a claim that Seme supports the same percentage of Go, JavaScript, or
all programming languages. Practical percentages should move only when real
programs, runtime contracts, and adversarial acceptance gates justify them.
The UAB-v1 and UPB-v1 percentages have different denominators and must never be
combined into one compatibility percentage.
