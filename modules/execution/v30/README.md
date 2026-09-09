# Core Execution Semantics v30

Version 30 adds runtime-keyed maps with exact key equality, canonical
zero-value lookup for missing keys, and immutable updates. Logical map meaning
does not expose storage or iteration order. The initial bounded realization
supports signed 64-bit keys and values with at most 512 entries.
