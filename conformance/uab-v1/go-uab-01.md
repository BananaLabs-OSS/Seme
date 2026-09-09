# Go UAB-01 evidence

`scripts/check-uab-v1-go-01.sh` is the complete evidence gate for the Go
UAB-01 cell. It proves the bounded capability directly; it does not claim
complete Go language, runtime, standard-library, or package compatibility.

- **Lift:** the Go provider parses and type-checks a three-file native package,
  resolves calls across files, and emits one validated canonical Program.
- **Native parity:** both the original and projected packages execute with the
  Go 1.26 native test toolchain and produce the expected nested call result.
- **Target parity:** the cumulative gate executes the same typed named-function
  and package-resolved call semantics through deterministic Wasm and pinned
  Pulp.
- **Projection round trip:** `goprojector` reads canonical Seme directly,
  emits gofmt-formatted native Go, and that Go re-lifts to byte-identical
  canonical meaning. JavaScript is not an intermediate representation.
- **Rejection:** unsupported canonical expressions and calls outside Program
  membership reject without guessed source or executable meaning.

The direct Go projector is intentionally bounded to ordinary named functions,
typed `int64`, `bool`, and `string` parameters and results, parameter reads,
closed calls, return blocks, and exact integer/text addition. Every other
canonical type, statement, or expression fails closed.
