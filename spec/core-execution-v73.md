# Core Execution v73: typed native binary operators

Core Execution v73 preserves language-owned division, remainder, shifts, and
bitwise operators as explicit typed expressions. Both operands, the operator,
and the result type remain visible while Go retains its exact numeric behavior.

Go projection restores native operator syntax. JavaScript and Lua views use
explicit `Seme.nativeBinary` and `Seme.native_binary` adapters rather than
conflating their number and bitwise models with Go's.
