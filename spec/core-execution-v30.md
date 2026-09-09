# Core Execution v30: runtime-keyed maps

Core v30 adds structural map types, empty maps, lookup, and immutable update.
Updating an existing key replaces its value; lookup of a missing signed `i64`
key returns canonical zero. Map equality and behavior are independent of
physical entry order. The bounded Wasm profile permits at most 512 entries.

The proof composes these operations through the existing generic `Fold` to
count repeated runtime values and query a caller-selected key.
