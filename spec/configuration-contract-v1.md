# Configuration and Initialization Contract v1

Module `...4000`, revision `...4001`, defines an immutable, language-neutral
configuration and initialization plan.

| Schema | ID | Meaning |
|---|---|---|
| ConfigurationGraph | `4010` | ordered fields, values, validation bindings, initializers, transitions, revision |
| ConfigurationField | `4011` | owner package, key, Execution type, optional pure default, required flag, exact source origin |
| ResolvedConfigurationValue | `4012` | field plus explicit/default value or capability-backed input |
| ResolutionOrigin | `4013` | closed code: explicit `0`, default `1`, capability input `2` |
| ValidationBinding | `4014` | optional field, validator Function, explicit order |
| InitializerUnit | `4015` | owner package, callable Function, predecessor initializers, explicit order |
| LifecycleState | `4016` | declared `0`, validated `1`, initializing `2`, initialized `3`, failed `4` |
| LifecycleTransition | `4017` | initializer, declared from/to states, explicit order |

Static explicit and default resolutions reference canonical typed values.
Default resolution must reference the field's declared default. Capability
input carries only an authorized Foundation Capability reference; its runtime
value and secrets are deliberately absent. Validators must prove defaults are
pure and type-correct, callable ownership and signatures, complete required
fields, dependency DAG/order, legal lifecycle transitions, and the graph's
content revision. Environment variables and host configuration syntax remain
provider/capability mechanics, not neutral semantics.
