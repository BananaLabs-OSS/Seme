# Lua UPB-06 evidence

Status: claimed 2026-09-12 by `scripts/check-lua-upb-06.sh`.

The cumulative gate passed all seven evidence classes. It authenticated an
eight-byte binary marker and nineteen-byte UTF-8 notice, emitted both under
their exact SHA-256 names, and bound Resource v1 to the preceding Project-v8
authority through Project v9.

Two original compositions were directory-identical. Canonical modular
projection preserved the resource declaration and source bytes exactly;
re-import reproduced canonical execution and an independent Project-v9 build
produced identical blobs and semantic resource report. The inherited native,
canonical, Wasm, and pinned-Pulp application observations remained green.

Content drift, traversal, wrong digest, wrong size, duplicate destination,
unknown metadata, symlink replacement, stale configured authority, and output
collision all rejected without partial publication.

The run used Node.js, Neovim, and cached Go 1.26.0 through process-local
environment variables. No software was installed and no persistent setting
changed.
