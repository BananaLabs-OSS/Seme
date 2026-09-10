# Ordered Transport Contract v1

This immutable contract describes bounded command ordering, opaque correlation,
exact duplicate responses, typed events, replay, and whole-frame host-port
authority. It contains no socket, HTTP, WebSocket, SSE, MessagePack, process,
clock, retry, or delivery mechanism.

The selected binary frame codec is `seme.ordered-transport.binary.v1`: a
little-endian `u32` body length (excluding the prefix), eight ASCII magic bytes
`SEMEOT01`, one `u8` frame-kind discriminator, then one exact canonical Pure
Value ABI v1 payload. The declared total length is exact; trailing bytes reject.
The complete frame is limited to 4,096 bytes.
