# Compiler

This directory is reserved for the human-authored semantic Seme compiler.

The compiler will be implemented above the frozen bootstrap chain. Its source
of truth will be canonical Kernel v1 entities and modules, not generated native
code or the temporary S0 presentation. Until that representation is frozen,
bootstrap compiler sources remain under `bootstrap/`.

The handoff criterion is strict: the semantic compiler must rebuild itself,
preserve conformance behavior and diagnostics, and let subsequent platform work
proceed without modifying assembly.

## Kernel wire validator

`kernel-wire-validator.s1` is the first compiler-layer program authored above
the self-hosted bootstrap. Checked S1 compiles it deterministically to
`kernel-wire-validator.k0`.

The current structural profile validates the Kernel v1 envelope, canonical
ULEB integers, bounds, strictly ordered parent/entity/field identities,
recursive lists and records, known value tags, nesting depth, and exact
end-of-file. With an output path it writes the validated canonical envelope
byte-for-byte, providing the preservation path for semantics the current
validator does not interpret. It does not yet interpret the Kernel meta-schema,
resolve semantic references/imports, migrate revisions, or produce structured
diagnostics.
