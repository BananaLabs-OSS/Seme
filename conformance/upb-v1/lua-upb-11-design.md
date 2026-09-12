# Lua UPB-11 acceptance design

Status: claimed 2026-09-12 by `scripts/check-lua-upb-11.sh`.

Lua UPB-11 applies neutral Patch v1, Live Language Service v1, and Project v14
to the cumulative native Lua project. Lua parsing, annotated semantic IDs,
native snapshot hashing, returned-table exports, JSON references, and Neovim
validation remain provider mechanics rather than additions to Seme Core.

The bounded edit addresses the stable `---@seme-id` of `InitializePolicy` and
projects `BuildPolicy` into its declaration, export key/value, and structured
configuration reference. Matching comment prose remains unchanged. Both source
revisions independently rebuild through Project v12 and target placement;
Project v14 binds their exact semantic transition and native transcript.

Acceptance requires monotonic complete-snapshot sessions, canonical identity
continuity, native behavioral parity, byte-identical twin publications, and
atomic rejection of forged IDs, stale bytes, simultaneous edits, collisions,
Lua keywords, metadata tampering, and existing destinations. Only
`scripts/check-lua-upb-11.sh` may claim this cell.
