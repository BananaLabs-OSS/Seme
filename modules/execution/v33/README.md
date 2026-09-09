# Core Execution Semantics v33

Version 33 adds two immutable, language-neutral collection operations:

- `SliceRemove(slice, index)` returns a fresh slice with exactly the indexed
  element removed. The index is zero-based and an out-of-bounds index rejects.
- `MapRemove(map, key)` returns a fresh map without the key. A missing key is
  a successful no-op with an observationally equal, non-aliased result.

Both operations preserve the relative order of surviving slice elements and
do not mutate or alias their input collection. This revision defines no Go
syntax, storage layout, or implementation strategy.
