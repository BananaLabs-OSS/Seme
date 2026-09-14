# Core Execution v42: runtime-owned type boundaries

Core Execution v42 generalizes the existing neutral `NativeType` boundary
across function parameters and results. A provider may retain a source-runtime
type only when that runtime owns its mechanics and the type is not a rejected
local semantic record.

The bounded Go provider uses the fully qualified `go/types` spelling as the
realization contract. Imported types such as `*net/http.Request` and Go
predeclared mechanics such as `int` can therefore remain typed at a Go-native
boundary. Local recursive or otherwise unsupported records remain rejected;
they are not relabeled as native merely to increase coverage.

This milestone adds no schema. It broadens the sound use of v41 `NativeType`.
Such values can cross compatible Go-native boundaries and appear in typed
signatures. Their fields, operators, memory behavior, and methods do not become
portable until separate semantics or explicit native operations describe them.
