# Product proof roadmap

Seme is developed through evidence-bearing layers:

```text
bootstrap and self-hosting
    -> trustworthy Kernel
    -> transactional semantic edits
    -> ordinary-project integration
    -> independent lowering
    -> package interoperability
    -> real Pulp workload
    -> WebAssembly portability
```

Completing a later layer does not waive an earlier layer's conformance gates.

## 1. Trustworthy Kernel v1 — complete

The structural Kernel v1 boundary is frozen. The executable evidence is tracked
in the [`Kernel v1 freeze evaluation`](kernel-v1-freeze-audit.md) and pinned in
[`KERNEL-V1-FREEZE.md`](../compiler/KERNEL-V1-FREEZE.md).

- Emit deterministic structured diagnostics with stable rule, revision, entity,
  field/list path, and structured argument identities.
- Complete schema/import, effects/capabilities, constraints/holes, provenance,
  refinement, unknown-field, and migration conformance.
- Freeze the canonical format, identities, validation behavior, and migrations.

Located Diagnostics 1 is the frozen structural validator: it preserves valid
inputs exactly, emits canonical reports for the invalid corpus, carries
trustworthy revision/entity identities, and uses the explicit unavailable
identity sentinel where malformed input cannot establish a field identity.
Richer semantic-module diagnostics remain versioned work above Kernel v1.

## 2. Transactional semantic editing

- Apply identity-based patches against an explicit starting revision.
- Commit the complete patch or leave canonical state unchanged.
- Preserve identity across rename, movement, formatting, and projection changes.
- Tombstone deleted identities and report conflicts deterministically.

## 3. Ordinary Go project proof

Status: complete for the explicitly scoped Provider Contract v1 Go profile.
The gate is `./scripts/check-provider-v1.sh`; broader Go constructs remain
separate provider-profile work rather than implied support.

The Go toolchain is an optional external ecosystem provider and is not part of
Seme's repository, trusted bootstrap, or independent reproduction path.
The first Go provider proves Provider Contract v1; it is not a claim of general
Go support. Later providers must reuse the same patch and reconciliation
contracts rather than introduce language concepts into the Kernel.

The milestone must demonstrate:

1. open an ordinary, non-Seme-authored Go repository;
2. import understood semantics and preserve unsupported source opaquely;
3. perform one identity-preserving semantic edit;
4. project the smallest valid change back into the ordinary repository;
5. preserve untouched bytes except where the native formatter contract requires
   a change;
6. pass the repository's original tests using its normal Go toolchain;
7. re-import the edited repository;
8. reconstruct the same stable semantic entities and relationships, with only
   the intentional semantic revision changed;
9. remove Seme and leave a normal usable Go repository.

Compilation success alone does not satisfy this milestone. The re-import check
is the semantic round-trip proof. Identity equality, relationship equality, and
unchanged-entity equality must be compared independently of source formatting.

## 4. Independent lowering

Complete for the frozen Core Execution v1 profile:

- the Go provider parses and type-checks the exact `int64` parameter-addition
  subset, then emits language-neutral canonical entities;
- the Seme-owned canonical interpreter executes those entities without Go at
  runtime and matches ordinary Go across normal, negative, zero, and signed
  overflow vectors;
- unsupported constructs reject explicitly instead of receiving guessed
  semantics.

This proves the independent-execution contract, not general Go lowering. The
next milestone broadens only as required by the package/application proof.

## 5. Package and application proof

Status: complete for Package Contract v1's scoped quota-policy profile.

- An ordinary Go package remains untouched and exposes a typed policy boundary.
- Canonical Package entities record the interface, explicitly empty dependency
  and effect sets, modular-integer runtime assumption, and exact fidelity with
  evidence.
- Core Execution v2 independently executes the package policy and matches Go,
  including signed overflow.

This establishes the package/application contract but is not yet the real Pulp
workload. The next proof should exercise a nonempty effect and dependency while
building the first Wasm portability plan.

## 6. WebAssembly portability proof

Status: canonical planning complete for the scoped logging profile; artifact
emission and Pulp execution remain.

- Target Contract v1 now analyzes a real `go:log` dependency and
  `observability.log` effect.
- The Wasm/Pulp plan records an adapted host-import Boundary and remains
  executable only when adaptation policy permits it.
- Exact-only policy produces impossible resolutions rather than dishonest
  exact fidelity.

- Analyze the transitive package closure against a versioned Wasm target
  contract.
- Resolve every requirement as exact, refined, adapted, emulated, embedded
  runtime, native island, or impossible.
- Emit and execute a faithful Wasm component when the plan permits it.
- Reject or expose mixed deployment when semantics cannot remain all-Wasm.
- Require no source changes made merely to cater to WebAssembly.

Additional projections, mechanics, ecosystems, and targets follow these proofs
and use the same provider, patch, target-contract, and conformance boundaries.
