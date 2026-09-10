# Project Contract v3

Project Contract v3 is the additive child of v2. It preserves the v1 semantic
snapshot and v2 source-inventory vocabulary, then adds neutral declarations
that bind a Package Contract v2 `PackageGraph` to a `SourceInventory` and the
result to a `ProjectSnapshot`.

V3 pins Package Contract `...b000@...b002` and Core Execution v35
`...9000@...9023`. The declarations contain no provider, filesystem, editor,
or application mechanics. Reproduce them with
`scripts/check-project-contract-v3.sh`.

No provider or emitter produces v3 instances yet. Instance validation,
content-revision algorithms, and build integration remain later milestones.
