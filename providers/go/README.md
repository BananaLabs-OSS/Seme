# Go provider proof

This is the first Provider Contract v1 dogfood fixture. It deliberately supports
only package-level functions, resolved references, and semantic rename. It is
not general Go support.

Parsing, name resolution, type checking, formatting, and native testing are
delegated to `go/parser`, `go/types`, `gofmt`, and `go test`. The adapter does
not handwrite those language facilities. Its current `refined` fidelity claim
covers stable function identity, compiler-resolved signatures/references, and
rename projection; it does not claim complete Go behavioral semantics.

Run the conformance test:

```sh
go test ./...
```

Or exercise the product loop against a disposable ordinary Go repository:

```sh
go run . import -root ./testdata/ordinary -out /tmp/seme-before.json
go run . rename -program /tmp/seme-before.json -name Greeting -to Welcome -out /tmp/seme-patch.json
go run . project -program /tmp/seme-before.json -patch /tmp/seme-patch.json -validate
go run . import -root ./testdata/ordinary -previous /tmp/seme-before.json -out /tmp/seme-after.json
go run . verify -before /tmp/seme-before.json -patch /tmp/seme-patch.json -after /tmp/seme-after.json
```

Use a copy of the fixture for the manual sequence because `project` intentionally
edits the native repository.
