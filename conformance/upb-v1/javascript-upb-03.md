# JavaScript UPB-03 evidence

Status: claimed 2026-09-11 by `scripts/check-javascript-upb-03.sh`.

The cumulative gate proves all seven evidence classes for the Project-v3
JavaScript fixture plus a bounded dependency closure. The adapter authenticates
one local ES-module edge and the exact npm package `@seme/checksum@1.2.3` from
an explicit offline tree. package.json and package-lock v3 agree on the exact
selection; SHA-512 integrity and an independent SHA-256 content digest bind the
external bytes. The resolver executes no install or lifecycle scripts and uses
no network access.

Two independent resolutions, Dependency-v1 artifacts, and Project-v4
authorities match byte-for-byte. Native projection preserves package.json,
package-lock.json, and other opaque units exactly; generated modules validate
and re-resolve the external closure offline. Floating ranges, lock selection
drift, source substitution, content tampering, malformed authority, and output
collisions reject without partial publication. The exact fixture retains
native/canonical/standalone-Wasm/pinned-Pulp parity.

This is a controlled exact npm subset, not general registry, semver, export-map,
workspace, lifecycle, or node_modules compatibility.
