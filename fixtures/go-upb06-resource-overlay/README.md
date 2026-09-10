# Go UPB-06 resource overlay

This cumulative overlay extends the materialized UPB-05 ordinary Go project. It
does not add filesystem or resource concepts to Seme Core.

The project owns two declared resources:

- `resources/notice.txt`: 26 UTF-8 bytes, including non-ASCII text and a final
  newline.
- `resources/marker.bin`: seven bytes (`00 ff 53 45 4d 45 0a`). The repository
  stores its checked base64 source because a textual patch cannot safely carry
  raw NUL and `ff`; the materializer creates the exact binary file.

`resources.json` fixes identity, source path, destination, media type, byte
length, and SHA-256 for each resource. `resource.Set` is an ordinary typed
value. Its pure `Validate` function returns errors 40 and 41 for invalid notice
and marker values. `service.ApplyConfiguredResource` validates the set before
delegating to the cumulative UPB-05 behavior.

The native corpus contains the prior 2,048 UAB11 observations and ten bounded
configuration observations with valid resources, preserving their behavior,
followed by eight resource observations. It reads the actual materialized
files and emits 2,066 deterministic request/expected JSONL pairs.

`adversaries/` reserves strict negative inputs for the future authenticated
resource pipeline: traversal, missing source, digest mismatch, duplicate
destination, declared over-limit size, unknown fields, and a target used to
construct a symlink during an isolated gate. These are evidence fixtures, not
accepted declarations.

Run `scripts/check-go-upb-06-fixture.sh` for the native foundation gate.
