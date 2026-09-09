# Go Project Build v1

Go Project Build v1 is the first bounded source-project path into a canonical
Project Contract artifact. It is deliberately smaller than Useful Project
Bridge v1 and does not by itself claim a UPB cell.

The pipeline is:

```text
complete in-memory Go snapshot
  -> typed IncrementalSession lift and package metadata
  -> canonical G1
  -> frozen K0 G1 compiler
  -> independent Kernel wire validation
  -> Execution v35 instance validation
  -> Package v1 assembly and validation
  -> Project v1 assembly and composed validation
  -> one create-only canonical .seme artifact
```

The compiler boundary is authoritative: the Go implementation does not parse
or reinterpret G1. The project builder accepts compilation as an injected
operation; the repository conformance adapter invokes the pinned bootstrap,
`g1-compiler.k0`, and `kernel-wire-validator.k0` directly without a shell.
Only a current `accepted-valid` session result may compile. Retained last-valid
editor state can never be published for an invalid current snapshot.

The command is a bounded conformance adapter, not a production publisher. It
requires absolute paths, rejects symlinks and nonregular project entries,
enforces source/artifact size bounds, forbids output inside the source tree,
uses a compiler timeout, authenticates the three contract artifacts through
the contract catalog, and publishes one new artifact with create-only
same-filesystem linking. Existing output is never replaced.

The fixture contains three local packages:

```text
application -> policy -> model
```

Its gate proves native Go behavior, exact client-revision independence,
source-comment and same-package filename independence, and propagation of a
semantic leaf edit into Package, ProjectSnapshot, and artifact revisions. It
also runs the independent Execution, Package, Project, and snapshot validators.

Run:

```sh
./scripts/check-go-project-build-v1.sh
```

This proof does not yet include source classification, external pinned
dependencies, project projection/edit round trips, target execution from the
assembled Project envelope, JavaScript or Lua project providers, resources,
configuration, services, effects, placement, or a Workbench consumer. Those
remain requirements of the frozen UPB-v1 denominator.
