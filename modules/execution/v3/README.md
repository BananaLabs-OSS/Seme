# Core Execution Semantics v3

Revision 3 preserves every v2 entity and adds the minimum record semantics
needed for schema-derived application boundaries:

- `RecordType` and ordered `RecordField` declarations;
- `FieldRead` expressions;
- `RecordConstruct` expressions.

The module is emitted from the shared declarative generator in
`reference/go/executionmodule`. Frozen v2 is reproduced byte-for-byte by the
same generator, preventing the extension from rewriting previous semantics.
