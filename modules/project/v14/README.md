# Project Contract v14

Project v14 (`...e000@...e035`, parent `...e032`) adds the compact
`ReconciledProjectRevision` (`e036`). It binds exact prior and resulting
Project-v13 snapshots, one Patch-v1 transaction, a monotonic client revision,
prior/result content revisions, and an independently observed native-validation
transcript digest.

The revision imports exact Project v13 (`...e000@...e032`), Patch v1
(`...5000@...5001`), and Live Language Service v1 (`...d000@...d001`). It does
not duplicate their schemas or cumulative imports. Rejected and incomplete
editor states remain language-service results and cannot become Project-v14
authority.

Source-language projection, formatting, native build/test execution, conflict
resolution, filesystem publication, targets, and downstream products remain
provider/runtime concerns outside this contract.
