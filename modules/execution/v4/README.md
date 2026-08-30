# Core Execution Semantics v4

Revision 4 preserves every v3 identity and adds the minimum variable-width and
fallible-value semantics required by application providers:

- `StringType` for Unicode text values;
- `BytesType` for opaque byte sequences;
- `ResultType`, `ResultOk`, and `ResultError`.

Concrete encodings remain target mappings. The canonical entities do not
contain pointers, byte offsets, UTF encodings, Wasm layouts, or Pulp ABI data.
