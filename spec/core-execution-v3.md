# Core Execution Semantics v3

Core Execution v3 is the first additive extension of the frozen execution
module. It preserves every v2 identity and adds four schemas:

| Identity | Schema |
|---|---|
| `...9030` | `RecordType` |
| `...9031` | `RecordField` |
| `...9032` | `FieldRead` |
| `...9033` | `RecordConstruct` |

Record fields have stable identities, names, referenced types, and explicit
indices. The index determines semantic field order; a source-language memory
layout does not leak into the canonical program. A target mapping derives its
own concrete layout and records the fidelity of that mapping.

The shared declarative generator emits both frozen v2 and v3. The v2 output is
checked byte-for-byte, proving that adding record semantics did not rewrite the
previous module.
