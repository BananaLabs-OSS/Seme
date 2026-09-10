# Go UPB-03 dependency-closure fixture

This bounded fixture has three internal semantic packages and one ordinary,
direct, version-pinned ecosystem requirement. The dependency is downloaded
from the repository-local offline GOPROXY and authenticated by `go.sum`. The
accepted `go.mod` has no `replace` or `exclude` directive.

UPB-03 proves dependency selection, source acquisition, integrity, and
repeatable offline resolution. The current bounded Go semantic provider still
rejects nonstandard external imports, so the dependency is deliberately not
called by these packages. Their native behavior remains local. Typed behavior
through an ecosystem-package boundary belongs to UPB-04 after that semantic
contract exists; treating a native-only call as already liftable would
overstate current support.
