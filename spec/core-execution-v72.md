# Core Execution v72: typed native pointer dereference

Core Execution v72 preserves language-owned pointer dereference as an explicit
typed expression. Its operand and result type remain visible in the canonical
graph while Go retains authority over pointer and memory behavior.

Go projection restores native dereference syntax. JavaScript and Lua views use
explicit `Seme.nativeDereference` and `Seme.native_dereference` adapters rather
than pretending those runtimes share Go pointer mechanics.
