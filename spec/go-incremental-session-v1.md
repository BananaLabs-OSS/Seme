# Go Incremental Session v1

The Go incremental session is an editor-facing provider service over complete
in-memory package snapshots. It does not read or write project files.

Each request contains:

- a strictly increasing unsigned client revision;
- a stable package path;
- a complete map of document paths to in-memory content.

The session deterministically hashes paths and contents, parses every non-test
Go document, type-checks the package with the native Go checker, and lifts each
supported package function through the existing compositional expression and
structured-body lifter. Source mappings connect stable semantic function
identities to document byte ranges and line/column locations.

Results use three dispositions:

- `accepted-valid`: the snapshot parsed, type-checked, and produced at least one
  supported canonical declaration;
- `accepted-invalid`: the revision advanced, but parsing or typing failed, or no
  supported declaration could be lifted;
- `rejected-stale`: the revision was zero or no newer than the latest accepted
  snapshot; session state did not change.

An accepted invalid snapshot returns located diagnostics and retains the last
valid canonical graph, canonical revision, and source mappings. This permits an
editor to remain runnable while a user is midway through an incomplete edit.
Unsupported but well-typed declarations produce located warnings; supported
declarations may still form a valid canonical subset. Unsupported source is not
inserted into the canonical graph or rewritten.

The bounded v1 lift accepts package functions with `int64` and `bool`
parameters, one `int64` or `bool` result, one return statement, and the current
Core v12 compositional expression vocabulary. It does not yet resolve sibling
in-memory packages, imported module dependencies, methods, locals, effects,
multiple returns, or build-tag variants.

The session API is safe for concurrent callers, but revision ordering—not
arrival time—determines acceptance. Equal snapshots with equal package path and
revision produce byte-identical canonical graphs and content digests.

The conformance gate feeds the session's emitted graph—not a parallel fixture
lift—through G1 compilation, Kernel and Foundation validation, the certified
pure-function lowerer, and standalone Wasm execution before running the full
Core v12/Pulp gate.

Live Language Service v1 remains the provider-neutral per-document transport
and state-transition contract. This package-snapshot session is a Go adapter
above that boundary; it does not redefine per-document digest or revision
semantics and can be connected by a transport that assembles complete package
snapshots from accepted document states.
