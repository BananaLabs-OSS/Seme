# Durable State Contract v1

Durable State v1 (`...8000@...8001`) describes versioned state families,
typed versions, pure validators and migrations, and capability-backed storage
port metadata. Load and atomic compare-exchange are declared Foundation effects;
filesystem and database mechanics remain provider concerns.

This metadata contract does not certify runtime token opacity, load/CAS trace
ordering, retry behavior, or atomic storage realization. Those properties need
separate runtime evidence from a capability-authorized DurablePort.
