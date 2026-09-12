# Lua UPB-01 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-01.sh` on
2026-09-12.

Lua UPB-01 applies the existing language-neutral source discovery, Package v1,
Project v1, Project v2, detached source, and atomic round-trip machinery to the
complete two-file cumulative Lua UAB-11 application. Lua parsing, annotations,
projection syntax, and native execution remain provider/target mechanics and
do not enter Seme Core.

Discovery records project identity, relative paths, SHA-256 digests, Neovim Lua
toolchain description, provider revision, and explicit tracked, ignored,
generated, vendored, and opaque classifications. Two discoveries must be byte-
identical. Ambiguous filesystem entries and symlinks reject.

A Lua-owned manifest adapter reads canonical meaning and derives the sole entry
function's exact identity, name, parameter/result types, and owned effect. It
does not infer these from filenames or hard-code this project. Neutral project
assembly and source inventory then bind the canonical application and complete
native source snapshot.

Projection emits ordinary Lua semantic source into a fresh project while
removing the replaced tracked files and preserving every opaque, generated,
ignored, and vendored byte. Original and projected projects execute the same
2,048-command corpus under Neovim and projected source re-lifts to identical
canonical G1 bytes.

Only `scripts/check-lua-upb-01.sh` may claim this cell. It reruns the complete
Lua UAB-11 source/target proof, the reusable project packages, manifest tests,
and source-authority gate. Digest drift, symlinks, malformed graphs, denied
effects, and destination collisions must publish no partial result.

This cell does not yet claim Lua module/package topology, LuaRocks resolution,
runtime embedding, arbitrary Lua projects, or Workbench.
