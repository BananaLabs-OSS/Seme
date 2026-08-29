# Kernel v1 freeze evaluation

Kernel v1 is complete at its structural boundary. The earlier audit mixed two
layers: the universal wire kernel and the candidate semantic-foundation module
represented through it. Freezing every future foundation schema into the wire
validator would contradict the small-kernel architecture.

Run the evidence with:

```sh
./scripts/check-kernel-v1.sh
./scripts/check-kernel-v1.sh --reproduce
```

## Frozen gate matrix

| Gate | State | Evidence |
|---|---|---|
| Bootstrap self-description and validator | pass | Seven-entity Module/Schema/Field bootstrap validates; the canonical located validator executes independently. |
| External module without kernel modification | pass | `external-counter` is accepted and preserved without changing the validator. |
| Unknown-field preservation | pass | Counter's unknown field round-trips byte-for-byte. |
| Structural rejection | pass | Noncanonical encoding, ordering, duplicate identities, dangling references, malformed values, and incomplete reserved bootstrap relationships reject. K0 separately rejects malformed control flow. |
| Deterministic diagnostics | pass | Every frozen rejection returns 65 and emits a canonical report with deterministic structural location and arguments. |
| Execute/lower compiler | pass | Frozen Core 1 self-reproduces; the located validator rebuilds byte-for-byte from its checked G1 projection and canonical graph. |
| Obsolete migration | pass | The v0 envelope migrates to canonical v1, validates, and preserves its entity payload. |

## Deliberately above Kernel v1

The candidate semantic-foundation module still needs generic validation for
cardinality, value shapes, imports, schema versions, effects/capabilities,
constraints, provenance, refinements, and richer diagnostic paths. These are
important Seme milestones, but they are versioned module semantics rather than
hard-coded wire concepts.

This separation lets Kernel v1 remain frozen while those facilities reach
their own 100% conformance gates.
