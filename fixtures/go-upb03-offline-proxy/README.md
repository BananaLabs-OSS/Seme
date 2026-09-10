# Go UPB-03 offline proxy

This directory is a standard file-based Go module proxy for the fixture's one
ecosystem dependency, `example.test/seme/checksum@v1.2.3`. The accepted
project's `go.mod` contains no `replace` or `exclude` directive.

The module source is retained at
`fixtures/go-upb03-external-checksum-v1.2.3`. The proxy zip uses the required
top-level module/version directory and fixed 2026-01-01 UTC entry timestamps.
Its SHA-256 is:

```text
b8c1fa6b91ffd1c51b765506ac14b80210a4019d75892e51782596d0f994910e
```

The acceptance gate uses a fresh `GOMODCACHE`, `GOPROXY=file://...`, and
`GOSUMDB=off`; integrity remains enforced by the committed `go.sum` hashes.
