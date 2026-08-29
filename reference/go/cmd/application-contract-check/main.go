// Command application-contract-check independently verifies the canonical
// package boundary emitted for the quota proof.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/goprovider"
	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 3 {
		fatal("usage: application-contract-check PROGRAM.seme MANIFEST.json")
	}
	graph, err := wire.Read(os.Args[1])
	check(err)
	manifest, err := goprovider.ReadManifest(os.Args[2])
	check(err)
	var function wire.ID
	for _, declaration := range manifest.Declarations {
		if declaration.Name == "Admit" {
			function, err = wire.ParseID(declaration.ID)
			check(err)
		}
	}
	must(function != (wire.ID{}), "Admit identity missing")
	functionEntity, ok := graph.Entities[function]
	must(ok && functionEntity.Schema == identity(0x9011), "Admit is not canonical Function")

	packages := bySchema(graph, 0xb010)
	interfaces := bySchema(graph, 0xb011)
	runtimes := bySchema(graph, 0xb013)
	mappings := bySchema(graph, 0xb014)
	must(len(packages) == 1 && len(interfaces) == 1 && len(runtimes) == 1 && len(mappings) == 1, "package evidence cardinality mismatch")
	pkg := packages[0]
	must(string(field(pkg, 0xb100).Bytes) == "example.com/seme-quota-proof", "package name mismatch")
	must(string(field(pkg, 0xb101).Bytes) == manifest.Revision, "package revision mismatch")
	must(len(field(pkg, 0xb102).List) == 1, "exported interface missing")
	must(len(field(pkg, 0xb103).List) == 0, "dependencies are not explicitly empty")
	must(len(field(pkg, 0xb104).List) == 0, "effects are not explicitly empty")
	must(len(field(pkg, 0xb105).List) == 1, "runtime assumption missing")
	must(len(field(pkg, 0xb106).List) == 1, "fidelity mapping missing")

	iface := interfaces[0]
	must(string(field(iface, 0xb110).Bytes) == "Admit", "interface name mismatch")
	must(field(iface, 0xb111).Reference == function, "interface does not reference stable Function")
	parameters := field(iface, 0xb112).List
	must(len(parameters) == 3, "interface parameter types mismatch")
	for _, parameter := range parameters {
		entity, exists := graph.Entities[parameter.Reference]
		must(exists && entity.Schema == identity(0x9010), "interface parameter is not IntegerType")
	}
	resultType, ok := graph.Entities[field(iface, 0xb113).Reference]
	must(ok && resultType.Schema == identity(0x9020), "interface result is not BooleanType")
	must(string(field(runtimes[0], 0xb130).Bytes) == "integer.i64.modular", "runtime assumption mismatch")
	must(field(mappings[0], 0xb140).Reference == function && field(mappings[0], 0xb141).Reference == function, "fidelity mapping identity mismatch")
	must(field(mappings[0], 0xb142).Unsigned == 0, "fidelity is not exact")
	must(len(field(mappings[0], 0xb143).Bytes) != 0, "fidelity evidence missing")
}

func bySchema(graph wire.Envelope, low uint64) []wire.Entity {
	want := identity(low)
	var out []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == want {
			out = append(out, entity)
		}
	}
	return out
}

func field(entity wire.Entity, low uint64) wire.Value {
	value, ok := entity.Fields[identity(low)]
	must(ok, "field %x missing", low)
	return value
}

func identity(low uint64) wire.ID {
	var value wire.ID
	value[14] = byte(low >> 8)
	value[15] = byte(low)
	return value
}

func must(ok bool, format string, args ...any) {
	if !ok {
		fatal(fmt.Sprintf(format, args...))
	}
}
func check(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func fatal(message string) {
	fmt.Fprintln(os.Stderr, "application-contract-check:", message)
	os.Exit(65)
}
