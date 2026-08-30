# Application Wire v2

Application Wire v2 is a bounded target mapping derived from Core Execution v4
records. It is not canonical Seme storage.

The quota request header contains three little-endian i64 values followed by
two little-endian u32 lengths. UTF-8 subject bytes and opaque evidence bytes
follow the 32-byte header. Each variable field is limited to 4096 bytes and the
total request length must exactly match the header.

The success response contains:

```text
u8  result_tag = 0
u8  accepted
u32 subject_length
u32 evidence_length
... subject UTF-8 bytes
... evidence bytes
```

The error response contains:

```text
u8  result_tag = 1
u32 message_length
... UTF-8 message bytes
```

The Wasm backend derives header offsets and sizes from canonical field order
and types, emits bounds/length checks, and copies the variable data. Both
`ResultOk` and `ResultError` are executed; the error branch is selected before
effects when the subject is empty.
