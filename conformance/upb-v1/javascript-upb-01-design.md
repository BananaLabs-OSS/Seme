# JavaScript UPB-01 acceptance design

Status: implementation in progress; unclaimed.

JavaScript UPB-01 uses the neutral Project-v2 source inventory and an explicit
ECMAScript policy. `.js` semantic sources are tracked; generated, ignored,
vendored, and opaque files remain byte-exact. The adapter records the pinned
Node identity and JavaScript provider semantic revision rather than inferring
either from the host.

The completed cell must bind this deterministic inventory to a validated
semantic Project snapshot, preserve the complete native tree, project tracked
ES modules from canonical meaning, run native Node tests offline, and re-lift
to exact canonical bytes. It must retain the UAB-01 native/canonical/Wasm/Pulp
behavior proof and reject unsafe paths, symlinks, classification ambiguity,
digest drift, malformed authority, and destination collisions atomically.

`scripts/check-javascript-upb-01-source.sh` is supporting discovery evidence;
it cannot claim the cell until the complete seven-class gate exists and passes.
