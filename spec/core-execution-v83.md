# Core Execution v83: unconditioned loops

Core Execution v83 lifts a Go `for { ... }` statement as an ordinary canonical
loop whose condition is the exact boolean value `true`. Existing canonical
branch semantics preserve explicit `break` and `continue` operations, so the
language-neutral graph does not retain Go's surface spelling.

This is the shared semantic concept “repeat until an explicit exit.” It maps
naturally to Go, JavaScript, and Lua projections and requires no canonical
schema change.
