# Go Incremental Session v1

The Go incremental session is an editor-facing provider service over complete
in-memory package or bounded module snapshots. It does not read or write
project files.

Each request contains:

- a strictly increasing unsigned client revision;
- an optional module path used to identify sibling local packages;
- a stable package path;
- an optional named entry declaration; and
- a complete map of document paths to in-memory content.

The session deterministically hashes paths and contents, groups every non-test
Go document by its package directory, and type-checks the requested package and
every declared sibling local package with the native Go checker. Standard
packages still resolve through the installed toolchain. It lifts supported
declarations through the existing compositional expression and structured-body
lifter. Source mappings connect stable semantic identities to document byte
ranges and line/column locations.

Results use three dispositions:

- `accepted-valid`: the snapshot parsed, type-checked, and produced at least one
  supported canonical declaration;
- `accepted-invalid`: the revision advanced, but parsing or typing failed, or no
  supported declaration could be lifted;
- `rejected-stale`: the revision was zero or no newer than the latest accepted
  snapshot; session state did not change.

An accepted invalid snapshot returns located diagnostics and retains the last
valid canonical graph, canonical revision, source mappings, resolved reference
occurrences, and package
metadata. This permits an editor to remain runnable while a user is midway
through an incomplete edit.
Unsupported but well-typed declarations produce located warnings; supported
declarations may still form a valid canonical subset. Unsupported source is not
inserted into the canonical graph or rewritten.

Each valid result also exposes a copy-safe, deterministically ordered neutral
view of all declared local packages. It records the selected root package,
direct local dependencies, and supported exported top-level functions with
their canonical identities and ordered parameter/result type identities. This
metadata comes from the same typed walk as the canonical graph; it is not
recovered by parsing G1 text and contains neither source bytes nor the client
revision. It is intended as provider evidence for a later Project Contract
instance, not as a claim about unsupported declarations or external packages.

Each typed identifier use that resolves to a supported local declaration is
reported as a half-open source occurrence carrying that declaration's stable
semantic identity. Occurrences are deterministically ordered and retained with
the last-valid result. They are navigation evidence only: unresolved or
unsupported names are omitted rather than matched heuristically.

The current bounded lift includes sibling local-package calls, named functions
and supported value-receiver methods, lexical locals and places, structured
control, supported records and collections, closures and bounded dispatch,
explicit transitions and fallibility, and declared observation effects from
the v35 application vocabulary. Fallthrough after a terminal `if` is
normalized to an explicit canonical else block. Support is still determined
per declaration; unsupported but well-typed declarations remain diagnosed and
omitted rather than guessed.

This is not a general Go project loader. It does not resolve arbitrary external
dependency ecosystems from the in-memory snapshot, evaluate build tags or
platform/file variants, reproduce `go generate`, cgo, assembly, plugins,
reflection, goroutine/channel semantics, or arbitrary standard-library and
runtime behavior. Multiple-result shapes outside the explicitly supported
Result/Option conventions also remain unsupported. The bounded local package forest
must not be described as general module compatibility.

When supplied the v14 execution module, a live snapshot containing text
parameters or a text result lowers through the exact variable-width
`seme.pure-abi/v2` profile. The acceptance gate executes a multibyte result
from the session-emitted canonical graph; this is the same graph path used for
scalar live edits rather than a separate source-shaped translation.

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
