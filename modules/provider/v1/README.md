# Provider Contract v1 module

`module.seme` is the canonical Provider Contract v1 schema authority.
`module.g1` is its checked readable construction projection, reproduced by the
independent `reference/go/cmd/provider-module` generator.

The module represents scoped provider profiles, native files, resolved source
occurrences, declarations, explicit identity evidence, opaque regions,
ingestion results, and projection reports. It contains no Go-specific syntax or
AST concepts.

The first adapter under `reference/go/goprovider` deliberately supports only
ordinary single-package Go modules with package-level functions. It uses the Go
parser and type checker, emits canonical Seme graphs, projects a committed
Patch v1 rename to proven resolved occurrences, validates in an isolated copy,
and re-ingests using explicit matching evidence.

Run the complete proof with:

```sh
./scripts/check-provider-v1.sh
```

