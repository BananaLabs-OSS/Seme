# Project Contract v10

Project v10 binds one exact Project-v9 resource snapshot to a Durable State v1
plan, a Source Presentation v1 manifest, and a content revision.

`DurableProjectSnapshot` (`e025`) retains `e250` for the Project-v9 snapshot,
`e251` for the durable plan, and `e252` for its 32-byte revision. It adds `e253`
for the presentation manifest. The revision must commit to both semantic
bindings, and validators must reject independently valid but mismatched
Project-v9, durable, or presentation inputs.

The revision imports Durable State `...8000@...8001` and Source Presentation
`...1000@...1001` exactly. Every Project-v9 declaration and pin remains
unchanged.
