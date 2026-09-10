# Project Contract v4

Project v4 additively binds the authenticated Project v3 graph snapshot to a
language-neutral Dependency v1 closure. `DependencyGraphSnapshot` references
the ProjectGraphSnapshot, DependencyClosure, and a content-derived revision.

V4 pins Dependency `f000@f001`, Package `b000@b002`, and Execution
`9000@9023`. It preserves every v3 declaration and contains no ecosystem- or
provider-specific resolution mechanics.
