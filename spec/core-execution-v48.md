# Core Execution v48: mixed neutral/native function products

Core Execution v48 applies the typed native product rule to ordinary Go
function signatures, not only to results returned by a native call.

A multi-result function may combine neutral Seme types with explicitly
Go-owned types such as target-sized `int` or `error`. The ordered `ProductType`
retains every item identity, and each native item has a corresponding
`NativeType` declaration. Application-owned records are not relabeled as Go
runtime types and remain unsupported until their meaning is represented.

This release adds no schema beyond v47.
