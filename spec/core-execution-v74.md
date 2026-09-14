# Core Execution v74: typed native indexed assignment

Core Execution v74 preserves language-owned indexed assignment as an explicit
statement. The collection, index, and assigned value remain independently
visible while Go retains its map, slice, and array mutation behavior.

Go projection restores native indexed assignment syntax. JavaScript and Lua
views use explicit `Seme.assignNativeIndex` and `Seme.assign_native_index`
adapters rather than conflating their collection mutation models with Go's.
