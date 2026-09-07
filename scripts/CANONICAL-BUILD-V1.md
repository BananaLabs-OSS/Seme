# Certified canonical build v1

`build-certified-canonical-v1.sh` is a canonical-first build adapter for the
bounded structured pure-function profile. It accepts already-produced canonical
Kernel wire; it has no source directory, language provider, ingestion manifest,
or native test input.

```text
scripts/build-certified-canonical-v1.sh \
  INPUT.seme EXPECTED_REVISION EXPECTED_SHA256 OUTPUT.wasm OUTPUT.json
```

Both identity pins are mandatory. `EXPECTED_REVISION` is the 128-bit revision
identity carried by the canonical envelope. `EXPECTED_SHA256` identifies the
exact canonical bytes supplied to the build.

The adapter:

1. Checks the byte digest before creating output staging files.
2. Runs the frozen structural validator.
3. Runs the Semantic Foundation validator and requires byte preservation.
4. Invokes the existing pure-function certifier and lowers only its immutable
   certificate.
5. Requires the derived ABI to report the pinned revision and input digest.
6. Independently checks the emitted artifact digest.
7. Publishes staged artifact and evidence files only after every check passes.

A stale digest or revision leaves existing outputs unchanged. The input is
never modified. Temporary paths are exact `mktemp` results and are removed on
normal exit or a handled signal.

Required executables are a POSIX shell, `go`, `sha256sum`, `awk`, `grep`, `cmp`,
`mktemp`, and the checked Linux-amd64 bootstrap executable in this repository.
The Go toolchain builds the existing reference backend; it is not used to read
or reconstruct source-language authority.

Run `scripts/check-certified-canonical-v1.sh` for the bounded conformance proof.
The check deliberately deletes its source-side fixture and provider products
before calling the canonical builder, compares the outputs with the checked
target artifacts, and verifies that a stale digest cannot replace prior output.

