# Ordinary Go provider proof fixture

This is an ordinary Go module with no Seme metadata or dependency. Provider
Contract v1 copies it to a temporary directory, imports package-level function
semantics, renames `Greeting` through its semantic identity, runs `go test
./...`, and re-imports it to prove identity recovery.

