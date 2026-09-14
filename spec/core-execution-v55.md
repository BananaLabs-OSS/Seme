# Core Execution v55: typed native package state

Core Execution v55 permits package-level Go variables whose internal type is
not yet neutral to remain explicit typed native binding reads.

The binding identity uses its fully qualified package and declaration name; its
value carries the exact Go type identity. Reads remain observable native
operations, so Seme does not mistake mutable package state for a neutral
constant or copy it into Core. Surrounding control flow can still be canonical
and projectable. No schema is added beyond v54.
