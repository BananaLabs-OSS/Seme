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

The Wasm backend derives header offsets and sizes from canonical field order
and types, emits bounds/length checks, and copies the variable data. This proof
executes only `ResultOk`; tag 1 and a canonical ResultError payload are reserved
until the source profile includes a real error-producing branch.
