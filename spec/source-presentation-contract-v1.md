# Source Presentation Contract v1

Source Presentation v1 separates source spelling from executable meaning.
Core Execution remains the authority for a type; `TypeAliasPresentation`
records that a source view intentionally names that type through an alias.

`PresentationManifest` (`1010`) contains an exact Project-v9
`ResourceProjectSnapshot`, a deterministic list of `TypeAliasPresentation`
records, and a 32-byte content revision. `TypeAliasPresentation` (`1011`)
contains an owning Package-v4 package, stable UTF-8 name, closed visibility,
target Execution type, authenticated Project-v8 `SourceUnit`, Package-v4
`Origin`, and referenced Package-v4 import bindings. `PresentationVisibility`
(`1012`) is closed to package/non-exported (`0`) and public/exported (`1`).

Alias entity identities and the manifest revision must commit to every field.
An instance validator must reject duplicate owner/name pairs, invalid UTF-8,
unknown visibility codes, targets absent from the bound project, origins that
do not match the named source unit, foreign owners/bindings, unsorted or
duplicate lists, stale revisions, and orphan entities.

The contract contains no Go, JavaScript, editor, filesystem, or projection
behavior. Language providers own syntax recognition; projectors consume only
validated presentation metadata.
