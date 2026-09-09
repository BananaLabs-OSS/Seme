# Project Contract v2

Project Contract v2 is revision `...e002` of module `...e000`, with Project
Contract v1 revision `...e001` as its sole parent. The Module declaration is
version 2. It preserves every v1 export and the exact Package v1 and Execution
v35 import pins.

V2 adds five neutral schemas:

- `ToolchainProfile` (`...e012`) explicitly records language, toolchain,
  interpretation profile, and its immutable semantic revision.
- `SourceClassification` (`...e013`) represents the closed classification.
- `PreservationMode` (`...e014`) represents preservation requirements.
- `SourceUnit` (`...e015`) records normalized relative path, content digest,
  byte size, classification, preservation mode, and ToolchainProfile.
- `SourceInventory` (`...e016`) has its own inventory revision, binds an
  ordered toolchain/source inventory to a ProjectSnapshot, and separately
  records that snapshot's 32-byte semantic revision.

`source_unit.classification` is a closed unsigned enumeration:

- 0: tracked
- 1: ignored
- 2: generated
- 3: vendored
- 4: opaque

All other values are invalid. The structural module declares the unsigned
code inside a distinct SourceClassification entity; an instance validator must
enforce the closed range.

PreservationMode codes are closed as 0 byte-exact, 1 semantic-projection, 2
regenerable, and 3 reference-only.

SourceInventory is a separate non-executable envelope. It is not embedded in
the semantic Project artifact. Its revision may change for filenames,
comments, ignored files, generated files, or bundle organization while the
bound ProjectSnapshot semantic revision remains unchanged. SourceUnit stores a
content digest, never raw source bytes; those bytes remain in digest-addressed
bundles outside canonical semantic entities.

The new schema identities occupy `...e012` through `...e016`; their fields use
`...e120` through `...e164`. They introduce no source-language, editor,
workspace, or application-specific mechanics.

Revision, entity, and module identities are role-scoped by the semantic-module
registry, so revision `...e002` may coexist with the v1-local import entity
`...e002`. Like v1, this contract contains schema references supplied by exact
imports. Its isolated wire therefore is not a closed Kernel graph; Foundation
validation plus import-pin and registry checks are authoritative here.
