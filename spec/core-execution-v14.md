# Core Execution Semantics v14

Core Execution v14 freezes the first variable-width UTF-8 runtime ABI over the
unchanged v13 canonical vocabulary. It adds no schemas. Canonical text is its
unique valid UTF-8 scalar encoding; no normalization is performed.

## Pure ABI v2

Any pure-function signature containing a string uses `seme.pure-abi/v2` and
provider `seme.function-v2`. ABI v1 and every scalar-only artifact remain
unchanged.

The fixed request header follows canonical parameter order. Boolean and i64
fields retain their v1 encodings. Each string occupies an eight-byte
little-endian `{offset,length}` descriptor. String payloads immediately follow
the header in parameter order: every offset must equal the running payload
cursor, and the final cursor must equal the exact request length. Gaps,
overlap, aliasing, trailing bytes, overflow, and truncation reject. Empty text
has length zero at the current cursor.

Each text value is limited to 4096 bytes and a request to 7160 bytes. Results
are raw UTF-8 bytes. The ABI evidence records the fixed header size, maximum
request size, variable-payload property, descriptor offsets and encodings, and
the existing canonical-program and artifact provenance.

Stable call failures are: `2` malformed request/header length, `3`
noncanonical Boolean, `4` invalid descriptor/order/bounds, `5` malformed UTF-8,
and `6` result allocation or size failure.

## Runtime realization

The capability-free Wasm cell validates UTF-8 with a generated finite-state
table that rejects overlong forms, surrogate encodings, invalid continuations,
out-of-range scalars, and incomplete sequences. Internal strings are packed
pointer/length values. Concatenation allocates bounded temporary storage and
uses Wasm bulk-memory copying; equality compares runtime bytes. Results are
copied to stable response scratch before temporary allocator state is rewound,
preserving the Pulp request lifetime and repeated-call behavior.

The JavaScript conformance oracle rejects lone UTF-16 surrogates before
`TextEncoder`, since its replacement behavior would otherwise erase a source
difference.

Run `./scripts/check-execution-v14.sh` for native, oracle, canonical,
standalone-Wasm, pinned-Pulp, malformed-input, lifetime, and predecessor
compatibility evidence.
