# Core Execution v58: typed native collection length

Core Execution v58 preserves `len` over collections whose mechanics remain
owned by Go.

The collection is materialized at its exact compile-time Go type, evaluated
once, and passed to an explicit native length operation returning Go `int`.
Lengths over neutral Seme arrays and slices retain their existing canonical
representation. This does not import Go map, channel, string-rune, or
application collection mechanics into Core. No schema is added beyond v57.
