# JavaScript UPB-03 acceptance design

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-03.sh`.

JavaScript UPB-03 extends Project-v3 with the existing neutral Dependency-v1
closure and Project-v4 binding. npm remains an ecosystem adapter; package.json,
package-lock v3, semver syntax, registry URLs, integrity strings, and Node module
layout do not become Core semantics.

## Bounded closure

The cumulative project retains its authenticated local application-to-math
dependency and declares one exact npm dependency. The fixture uses a controlled
offline package artifact and lock entry so resolution can be repeated without
network access. The adapter records requested identity/range, exact selected
version, SHA-512 integrity bytes, canonical source identity, dependency kind,
and applicable npm metadata.

The root package manifest and lockfile are inventoried byte-exact units. The
resolved package content is independently hashed; neither the lockfile nor an
installed `node_modules` directory is trusted alone. Reordered JSON object keys
may preserve dependency meaning while any changed requirement, selection,
integrity, source, or applicable metadata changes or invalidates the closure.

## Rejection boundary

Missing lock entries, floating or non-exact selections, range/selection
mismatch, unsupported protocols, integrity mismatch, undeclared installed
packages, dependency confusion between local and npm identities, substitutions,
unexpected transitives, symlinks, path escapes, and network-dependent
resolution reject before publication. npm lifecycle scripts are never executed
by this bounded resolver.

Only `scripts/check-javascript-upb-03.sh` may claim the cell. It must execute
the complete UPB-02 gate, resolve twice offline, bind Project-v4, retain exact
native/canonical/Wasm/Pulp behavior and native projection, and cover all seven
evidence classes plus dependency-specific adversaries.
