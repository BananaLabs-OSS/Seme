# Project Contract v8

Project v8 (`...e000@...e00a`, parent `...e009`) adds
`FullConfiguredProjectSnapshot` (`e023`). One snapshot binds the exact source
inventory, dependency closure, Package v4 graph, Execution v36 program, and
Configuration v3 bound graph with a 32-byte content revision.

The contract intentionally describes one complete source snapshot. It does not
combine an older v35 project artifact with a separately lifted v36 program.
