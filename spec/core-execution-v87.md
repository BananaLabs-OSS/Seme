# Core Execution v87: increment and decrement updates

Core Execution v87 lifts standalone increment and decrement statements as
ordinary updates to mutable places. Canonical `i64` values use neutral addition
or subtraction by one; other Go numeric types use an explicit typed native
compound operation.

No Go-shaped increment node is added to core, and no canonical schema change is
required.
