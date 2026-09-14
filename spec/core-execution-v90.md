# Core Execution v90

Core Execution v90 preserves ordinary parallel assignments whose targets mix
local bindings and Go fields.

Every selector receiver is materialized before every right-hand side. All
right-hand sides are then materialized before mutations execute in source
order, retaining Go's parallel-assignment evaluation semantics. Field writes
remain explicit native field assignments.

No canonical schema changed. Older module artifacts and identities remain
byte-for-byte reproducible.
