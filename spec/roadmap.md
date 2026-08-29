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

## 1. Trustworthy Kernel v1

- Emit deterministic structured diagnostics with stable rule, revision, entity,
  field/list path, and structured argument identities.
- Complete schema/import, effects/capabilities, constraints/holes, provenance,
  refinement, unknown-field, and migration conformance.
- Freeze the canonical format, identities, validation behavior, and migrations.

The diagnostic report shape now has canonical compile/validate/round-trip
fixtures, and the post-freeze validator emits a report for every current
rejection path. Diagnostic transport and broad stable rule classification are
complete. The Kernel gate remains open until entity and field/list locations
are refined and covered by conformance fixtures.

Located Diagnostics 1 is complete: its canonical validator preserves valid
inputs exactly, emits canonical reports for the current invalid corpus, carries
trustworthy revision/entity identities, and records field-level failures with
the explicit unavailable-identity sentinel. Exact field identity plus nested,
list, and reference provenance remain part of the open Kernel gate.

## 2. Transactional semantic editing

- Apply identity-based patches against an explicit starting revision.
- Commit the complete patch or leave canonical state unchanged.
- Preserve identity across rename, movement, formatting, and projection changes.
- Tombstone deleted identities and report conflicts deterministically.

## 3. Ordinary Go project proof

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

- Lower the supported canonical subset without treating Go as authoritative.
- Keep unsupported constructs explicit as opaque/provider dependencies.
- Compare behavior with the original Go execution as conformance evidence.

## 5. Package and application proof

- Represent one normal package boundary, including effects, runtime assumptions,
  dependencies, typed interfaces, and fidelity.
- Run one useful Sessions/Pulp component through the complete path.

## 6. WebAssembly portability proof

- Analyze the transitive package closure against a versioned Wasm target
  contract.
- Resolve every requirement as exact, refined, adapted, emulated, embedded
  runtime, native island, or impossible.
- Emit and execute a faithful Wasm component when the plan permits it.
- Reject or expose mixed deployment when semantics cannot remain all-Wasm.
- Require no source changes made merely to cater to WebAssembly.

Additional projections, mechanics, ecosystems, and targets follow these proofs
and use the same provider, patch, target-contract, and conformance boundaries.
