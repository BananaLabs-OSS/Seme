# Project Contract v6

Project v6 uses revision `...e008`; `...e006` and `...e007` are already import
entity identities. It adds `ConfiguredProjectSnapshot` (`e021`) with fields:

- `e210`: exact Project v5 `CompleteProjectGraphSnapshot`;
- `e211`: exact Configuration v1 `ConfigurationGraph`;
- `e212`: immutable content revision.

This binds configuration and initialization evidence to the exact project
without copying Execution values, callables, runtime state, or secrets.
