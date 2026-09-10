# Project Contract v8

Project v8 adds `FullConfiguredProjectSnapshot` (`e023`) with required fields:

- `e230`: SourceInventory (`e016`)
- `e231`: DependencyClosure (`f010`)
- `e232`: CompletePackageGraph (`b029`) under Package v4
- `e233`: ExecutableProgram (`9015`) under Execution v36
- `e234`: BoundConfigurationGraph (`401f`) under Configuration v3
- `e235`: 32-byte content revision

Validators must prove all five components derive from the same complete source
snapshot and reject mixed revisions or independently lifted roots.
