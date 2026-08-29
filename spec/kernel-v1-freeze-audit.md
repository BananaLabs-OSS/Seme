# Kernel v1 freeze audit

This audit evaluates the candidate against the seven gates in
[`kernel-v1.md`](kernel-v1.md). Completion means executable canonical evidence,
not that a schema identity or intended behavior is documented.

Run the current evidence with:

```sh
./scripts/check-kernel-v1.sh
```

## Gate matrix

| Gate | State | Current evidence | Missing evidence |
|---|---|---|---|
| Self-schema and validator | partial | The validator is canonical Seme and the seven-entity Module/Schema/Field bootstrap slice validates. | Complete finite meta-schema; generic cardinality, `ValueShape`, constraint, and schema-version validation. |
| External semantic module | partial | `external-counter` is structurally accepted without changing the validator. | Counter expressed through Module, Import, Schema, and Field declarations and validated from those declarations. |
| Unknown-field preservation | pass | `external-counter` round-trips its unknown field byte-for-byte. | Retain this fixture in the frozen suite. |
| Reject malformed semantics | partial | Encoding, ordering, duplicate identity, dangling local reference, reserved bootstrap schema, and K0 control-flow fixtures reject. | Imported references, schema field shapes/cardinality/versions, effects, capabilities, and constraint failures. |
| Canonical deterministic diagnostics | partial | Every current rejection produces a canonical report with stable broad classification and revision/entity location where available. | Exact field identity and nested record/list/reference paths; distinct stable semantic rules and structured arguments. |
| Execute/lower own compiler | pass | The canonical lowerer reproduces itself and rebuilds the compiler tools. | Retain byte-exact A/B/C evidence in the frozen suite. |
| Obsolete fixture migration | pass | The executable v0 envelope migration validates and preserves its payload. | Retain the migration fixture; schema migrations belong to versioned modules unless promoted into this gate. |

The current freeze state is therefore three passing gates and four partial
gates. Kernel v1 is not freeze-ready.

## Missing conformance groups

The following groups must exist before the candidate can be called complete:

1. **Complete meta-schema:** every Schema, Field, Module, Import, Constraint,
   Effect, Capability, Provenance, Refinement, DiagnosticRule, Diagnostic, and
   DiagnosticReport declaration is data in the checked Kernel module.
2. **Generic schema validation:** required/optional/many cardinality, declared
   value shapes, schema versions, unknown-version preservation, and stable
   validation order.
3. **Imports:** available imported identities resolve; unavailable imports are
   preserved without semantic certification; malformed module and minimum
   revision declarations reject.
4. **Authority semantics:** effects reference capabilities; module-required
   effects resolve; malformed effects and capabilities reject.
5. **Constraints:** a versioned constraint operation has defined deterministic
   semantics and emits its declared diagnostic rule. The current opaque
   `constraint.operation` bytes are not yet an executable language.
6. **Provenance and refinement:** field shapes, reference validity, revision
   identity width, and evidence requirements have positive and negative cases.
7. **Exact diagnostics:** envelope, entity, field, nested-record, list-index,
   and reference-target failures carry the precise semantic path and stable
   rule identity.

## Implementation order

Close the gates in dependency order:

```text
structured canonical validator authoring
-> complete finite meta-schema
-> generic shape/cardinality validator
-> imports and schema versions
-> effects/capabilities
-> constraint operation v1
-> provenance/refinement
-> exact diagnostic paths
-> frozen full-suite reproduction
```

The generic validator must be implemented in canonical Seme. A host-language
implementation may be a differential oracle or fixture generator, but cannot
satisfy the self-hosted freeze gate.

The current located validator is canonical and reproducible, but its checked G1
projection is a thousands-of-instructions graph lifted from bootstrap source.
Extending that graph by hand is not an acceptable continuing implementation
surface. Before adding the generic validator, Seme needs a structured canonical
function/control-flow projection that lowers through the existing canonical K0
module lowerer. That projection is part of completing Kernel v1, not a new host
implementation language.
