# Project Contract v4

Project module `e000`, revision `e004`, is the additive child of revision
`e003`. It adds schema `e019` (`DependencyGraphSnapshot`):

- `e190` references the Project v3 `ProjectGraphSnapshot` (`e018`).
- `e191` references Dependency v1 `DependencyClosure` (`f010`).
- `e192` stores the 32-byte content revision of the combined closure.

The module imports Dependency `f000@f001`, Package `b000@b002`, and Core
Execution `9000@9023`. Dependency resolution remains an independent semantic
module; Project v4 only authenticates which immutable closure belongs to the
project graph.
