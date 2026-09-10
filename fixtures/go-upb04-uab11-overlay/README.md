# Go UPB-04 UAB-11 overlay

The authoritative Go sources and 2,048 native vectors remain in
`fixtures/go-uab-11`. A conformance build copies that fixture and applies this
small module overlay. `SOURCES.sha256` pins every copied source and test, so a
materialized fixture is an ordinary self-contained Go project without silently
drifting from the UAB-11 corpus. The pinned offline dependency is deliberately
resolved and authenticated but is not called by the application.
