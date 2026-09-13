# Core Execution v37: neutral unit

Core Execution v37 adds the neutral `UnitType` and its sole `UnitValue`.

Unit represents completion without returned information. It is not Go's empty
result list, JavaScript `undefined`, Lua `nil`, a null pointer, Boolean success,
an empty tuple with observable arity, or a target ABI convention. Ecosystem
providers may map native no-result completion to Unit only when doing so
preserves all observable meaning claimed by that provider profile.

`UnitValue` explicitly references `UnitType`, keeping type certification
structural and target-independent. Effects performed before completion remain
separate semantic operations and are not encoded in Unit.

## Frozen identities

| Identity suffix | Declaration |
|---|---|
| `a06a` | `UnitType` |
| `a06b` | `UnitValue` |
| `a06b0` | `unit_value.type` |

Version 37 is strictly additive over v36.
