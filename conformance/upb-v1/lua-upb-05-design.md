# Lua UPB-05 acceptance design

Status: implemented and claimed by `scripts/check-lua-upb-05.sh` on
2026-09-12.

The bounded Lua project declares typed boolean, i64, and string configuration
fields with defaults; validates the limit; then initializes configuration,
policy, and application records in an explicit dependency order. Neutral
Configuration v3 owns the resolved fields and lifecycle graph, while Project
v8 binds it to Execution v36, Package v4, Dependency v1, and the complete
source inventory.

Lua `---@class` declarations become authenticated package members with exact
canonical record identities and source origins. The modular projector restores
those native annotations and their functions to their owning files. A neutral
project-manifest adapter now derives Package-v1 input from any conforming
project graph; the JavaScript entry point remains only a compatibility wrapper.

The sole gate inherits UPB-01 through UPB-04, reproduces the full configured
project twice, executes the initialization plan, projects and exactly re-lifts
the modules, and compares native, canonical, canonical-Wasm, and pinned-Pulp
behavior. Invalid lifecycle dependencies, ambient `os.getenv`, wrong boundary
types, authority tampering, and output collisions reject atomically.

This cell does not claim secrets, arbitrary environment ingestion, or general
Lua initialization behavior.
