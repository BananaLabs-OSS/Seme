# Configuration and Initialization Contract v2

Configuration v2 is additive revision `...4005` of module `...4000`, with v1
revision `...4001` as its parent. It preserves all v1 declarations and adds:

| Schema | ID | Meaning |
|---|---|---|
| RuntimeInput | `4018` | stable startup ABI identity, Execution type, optional Capability authorization; never a runtime value |
| ArgumentSourceKind | `4019` | closed codes: resolved field `0`, runtime input `1`, predecessor success payload `2`, static canonical value `3`, record construction `4` |
| ArgumentSource | `401a` | exact derived type and exactly one source selected by kind |
| BoundInitializerUnit | `401b` | v1 initializer plus complete ordered arguments |
| RecordAssembly | `401c` | exact RecordType plus every RecordField binding in canonical order |
| RecordMemberBinding | `401d` | RecordField and recursively typed source |
| InitializerArgument | `401e` | exact Function Parameter identity/index and source |
| BoundConfigurationGraph | `401f` | v1 graph, runtime inputs, bound initializers, content revision |

Arguments must match the callable's parameter identities and indices exactly.
A predecessor-success source must name a declared initializer dependency whose
callable result is a Result realization with the same success type. Recursive
record sources are acyclic and bounded. Runtime input identities are nonempty,
unique UTF-8; capability-backed inputs contain authorization but no secrets or
values. Static values remain canonical Execution values. No ambient host
environment mechanism is part of this contract.
