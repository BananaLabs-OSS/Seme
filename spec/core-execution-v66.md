# Core Execution v66: typed nearest branches

Core Execution v66 adds `NativeBranch`, recording its owning language,
operation, and target scope. The Go provider accepts unlabeled `break` and
`continue` and records the target as `nearest`. This preserves Go's own
nearest-breakable resolution: `continue` selects a loop, while `break` may
select a loop or switch. When a posted `for` or indexed `range` is lowered to
canonical `while`, Seme injects the synthesized post step before each owned
`continue`, preserving Go's iteration behavior.

The boundary is language-owned because `continue` availability and labeled
transfer rules differ across target languages. Go projection restores the
native keyword. JavaScript and Lua projections expose explicit runtime adapter
operations instead of silently changing control flow.

Labels, `goto`, and `fallthrough` remain rejected until the
canonical graph can identify their targets exactly. This milestone therefore
widens complete-project visibility without weakening fidelity claims.
