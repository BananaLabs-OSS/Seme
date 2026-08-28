# Compiler

This directory is reserved for the human-authored semantic Seme compiler.

The compiler will be implemented above the frozen bootstrap chain. Its source
of truth will be canonical Kernel v1 entities and modules, not generated native
code or the temporary S0 presentation. Until that representation is frozen,
bootstrap compiler sources remain under `bootstrap/`.

The handoff criterion is strict: the semantic compiler must rebuild itself,
preserve conformance behavior and diagnostics, and let subsequent platform work
proceed without modifying assembly.
