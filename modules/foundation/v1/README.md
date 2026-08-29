# Semantic Foundation Module v1

`module.seme` is the authoritative canonical graph for the complete Foundation
v1 declaration set. It contains 14 schemas, their field declarations, recursive
ValueShape and PathSegment records, and the distinct Foundation module identity
above frozen Kernel v1.

`module.g1` is the checked construction projection. The Go generator under
`reference/go/cmd/foundation-module` is a differential fixture generator, not
the authority.

Reproduction, hash, frozen-Kernel, and oracle checks run through:

```sh
./scripts/check-semantic-modules.sh
```
