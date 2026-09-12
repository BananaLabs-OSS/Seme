# Lua UPB-08 acceptance design

Status: claimed 2026-09-12 by `scripts/check-lua-upb-08.sh`.

Lua UPB-08 extends Project v10 with a bounded, provider-neutral ordered
transport. Ordinary Lua declares command and event records plus pure dispatch
and replay functions. The adapter resolves their package ownership, semantic
types, and callable identities; the contract contains no Lua syntax or fixture
names.

Ordered Transport v1 owns bounded sequencing, correlation, replay, whole-frame
ports, and placement requirements. Network protocols and socket behavior are
explicit provider mechanics and are not invented by Core. Project v11 binds the
exact Project-v10 and transport authorities.

Acceptance requires reproducible authorities; native Lua, canonical, Wasm, and
pinned-Pulp dispatch parity including i64 wraparound; neutral host and placement
tests; exact cumulative projection/re-lift; and atomic rejection of missing
owners or callables, aliased roles, duplicate kinds, undeclared provider fields,
stale Project-v10 authority, and output collisions. Only
`scripts/check-lua-upb-08.sh` may claim this cell.
