# Dependency Contract v1

This directory contains the generated language-neutral dependency-resolution
contract. `DependencyClosure` binds ordered root requirements to an immutable
resolved closure and a content revision.

The contract distinguishes local and ecosystem dependencies with the closed
`DependencyKind` codes `0` and `1`. Ecosystem identity, selected version,
integrity, source, transitive edges, and applicable key/value metadata are
explicit. It does not encode Go modules, npm, Cargo, filesystem discovery, or
any other package-manager-specific mechanism.

Reproduce and validate the artifacts with `scripts/check-dependency-v1.sh`.
