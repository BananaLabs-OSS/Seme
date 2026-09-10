# Project Contract v7

Project v7 is additive revision `...e009` of module `...e000`, with Project v6
revision `...e008` as its parent. It preserves every v6 declaration, imports
Configuration v2 (`...4000@...4005`), and adds:

| Schema | ID | Meaning |
|---|---|---|
| BoundConfiguredProjectSnapshot | `e022` | binds a Project v6 configured snapshot to an executable Configuration v2 bound graph and content revision |

Fields `e220`, `e221`, and `e222` reference the v6 snapshot, Configuration v2
`BoundConfigurationGraph`, and the binding's content revision respectively.
This layer authenticates the initialization binding without placing live
runtime inputs, capability values, or secrets in the project artifact.
