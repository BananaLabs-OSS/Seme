# Core Execution v65: typed native switch control

Core Execution v65 adds `NativeSwitch` and `NativeSwitchCase`. A switch records
its owning language, subject (or Unit for an expressionless switch), ordered
cases, case values, canonical case bodies, and an optional canonical default
body.

The boundary is deliberately language-owned. Go's one-time tag evaluation,
comparability rules, implicit case break, expressionless form, and multi-value
cases are not silently equated with JavaScript or Lua control mechanics. Go
projection restores ordinary switch syntax. JavaScript and Lua projections
expose an explicit runtime adapter containing the same ordered values and
callable bodies.

The Go provider accepts tagged and expressionless switches, multiple values per
case, optional defaults, and an explicit unlabeled terminal `break`. It rejects
initializers, fallthrough, labeled branches, and other control transfers until
their ownership and targets can be represented exactly.
