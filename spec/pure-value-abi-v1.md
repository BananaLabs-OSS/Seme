# Pure Value ABI v1

`seme.pure-value-abi/v1` is the type-derived boundary-layout foundation for a
future compositional pure-function ABI. It does not replace or alter
`seme.pure-abi/v1` or `/v2`, and therefore cannot change existing artifacts.

All fixed headers are packed bytes with no implicit host alignment. Offsets in
variable descriptors are unsigned little-endian 32-bit offsets from the start
of the complete request or response. Descriptor ranges must be in bounds,
must not overlap fixed headers or another payload, and must respect the stated
maximum. Inactive union bytes are zero, preventing unselected data from
acquiring observable meaning.

| Canonical type | Fixed representation |
|---|---|
| signed i64 | 8-byte little-endian two's-complement modular word |
| Boolean | one byte, exactly `0` or `1` |
| text | `u32 offset, u32 byte_length`; at most 4096 scalar UTF-8 bytes |
| bytes | `u32 offset, u32 byte_length`; at most 4096 opaque bytes |
| `Result<O,E>` | tag byte (`0` ok, `1` error), then `max(sizeof(O), sizeof(E))` union bytes |
| `Option<T>` | tag byte (`0` none, `1` some), then `sizeof(T)` bytes |

Selected Result and Option payloads recursively use their certified canonical
type layout. Variable payload bounds propagate outward. Unknown tags,
noncanonical Booleans or UTF-8, out-of-range/overlapping descriptors, nonzero
inactive bytes, missing type references, recursive type graphs, and depth over
32 reject.

The value-layout specification fixes representation and validation. The
separate bounded composite function profile below embeds those layouts only
after its graph, request decoding, response allocation, expression lowering,
standalone Wasm, and Pulp execution are certified together.

`seme.pure-composite-abi/v1` is that function-level envelope's reserved
metadata contract. It orders certified parameter layouts by canonical
parameter index, carries one result layout, limits each complete message to
7160 bytes, and exposes provider `seme.function-composite-v1`. Certification
of this metadata does not itself authorize Wasm emission; executable graph
certification remains mandatory.

The first executable composite reactor is intentionally a bounded semantic
profile. It certifies exactly one canonical function of type
`Option<Result<bytes,text>> -> Boolean`, eliminated by total `OptionMatch` and
`ResultMatch` nodes whose selected payloads are read through lexically scoped
`VariantBindingRead` nodes. Its leaf comparisons are canonical `BytesEqual`
and `StringEqual` operations against literals. This profile is an execution
proof for that shape, not a general recursive composite-expression lowerer;
other composite signatures and expression graphs must fail closed until a
later profile specifies and certifies them.

Run `./scripts/check-composite-runtime-v32.sh`. The gate compiles a
consumer-neutral canonical fixture, lowers it twice, compares Wasm and ABI
bytes, executes five valid and eight malformed standalone vectors, verifies
request immutability, and executes None, Ok, and Error cases through pinned
Pulp.
