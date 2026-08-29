# Patch Module v1

`module.g1` is the checked readable projection of the Patch Module v1 schema.
`module.seme` is its authoritative canonical Kernel v1 graph. The Go code under
`reference/` is a differential oracle, not the module authority.

Reproduce and validate the graph:

```sh
bootstrap/seme-k0-linux-amd64 compiler/g1-compiler.k0 \
  modules/patch/v1/module.g1 /tmp/patch-v1.seme
cmp modules/patch/v1/module.seme /tmp/patch-v1.seme
bootstrap/seme-k0-linux-amd64 compiler/kernel-wire-validator-located.k0 \
  /tmp/patch-v1.seme
```

The repository gate runs this reproduction plus the differential oracle suite:

```sh
./scripts/check-semantic-modules.sh
```
