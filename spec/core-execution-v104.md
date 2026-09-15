# Core Execution Semantics v104

Core Execution v104 completes Go empty struct-literal lifting. `Record{}` now
materializes every field through its typed zero value, just as omitted fields in
a keyed literal already do. This covers scalar, pointer, slice, map, and nested
record defaults without embedding source text.

The result uses existing canonical record construction and default-value nodes;
v104 therefore preserves the full v103 schema identity set.

Run `./scripts/check-execution-v104.sh` to reproduce the module, compiled
artifact, fixture lift, and native Go projection proof.
