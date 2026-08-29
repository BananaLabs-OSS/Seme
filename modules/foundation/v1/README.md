# Semantic Foundation Module v1

`module.seme` is the authoritative canonical graph for the complete Foundation
v1 declaration set. It contains 14 schemas, their field declarations, recursive
ValueShape and PathSegment records, and the distinct Foundation module identity
above frozen Kernel v1.

`module.g1` is the checked construction projection. The Go generator under
`reference/go/cmd/foundation-module` is a differential fixture generator, not
the authority.

`validator.seme` is the authoritative executable Foundation validator. Its
checked `validator.k0` is a derived execution artifact and `validator.s1` is
retained as readable bootstrap construction history. The validator consumes a
frozen-Kernel-valid envelope, validates every entity whose schema declaration
is available in that envelope, recursively checks lists and records, enforces
cardinality, schema versions, and reference-schema constraints, preserves
unknown future versions and unavailable schemas, and preserves successful
input byte-for-byte. The canonical graph lowers back to the checked K0 image
exactly.

Reproduction, hash, frozen-Kernel, and oracle checks run through:

```sh
./scripts/check-semantic-modules.sh
```
