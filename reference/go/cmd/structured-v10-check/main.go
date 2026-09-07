// Command structured-v10-check independently checks a generic v10 body graph.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 2 {
		fatal("usage: structured-v10-check PROGRAM.seme")
	}
	graph, err := wire.Read(os.Args[1])
	check(err)
	functions := bySchema(graph, 0x9011)
	blocks := bySchema(graph, 0x9080)
	returns := bySchema(graph, 0x9081)
	subtracts := bySchema(graph, 0x90a0)
	multiplies := bySchema(graph, 0x9090)
	literals := bySchema(graph, 0x9070)
	if len(functions) != 1 || len(blocks) != 1 || len(returns) != 1 || len(subtracts) != 2 || len(multiplies) != 1 || len(literals) != 1 {
		fatal("unexpected structured graph cardinality")
	}
	block := referenced(graph, functions[0], 0x9113)
	if block.Schema != identity(0x9080) {
		fatal("function body is not a Block")
	}
	statements := value(block, 0x9800).List
	if len(statements) != 1 || graph.Entities[statements[0].Reference].Schema != identity(0x9081) {
		fatal("block does not contain one Return")
	}
	returned := graph.Entities[statements[0].Reference]
	values := value(returned, 0x9810).List
	if len(values) != 1 {
		fatal("Return does not contain exactly one value")
	}
	root := graph.Entities[values[0].Reference]
	if root.Schema != identity(0x90a0) {
		fatal("Return root is not IntegerSubtract")
	}
	left := referenced(graph, root, 0x9a00)
	right := referenced(graph, root, 0x9a01)
	if left.Schema != identity(0x90a0) || right.Schema != identity(0x9090) {
		fatal("ordered nested operands were not preserved")
	}
	if value(literals[0], 0x9700).Unsigned != 2 {
		fatal("integer literal meaning was not preserved")
	}
	fmt.Println("Core Execution v10 ordered subtraction graph passed")
}

func bySchema(graph wire.Envelope, schema uint64) []wire.Entity {
	want := identity(schema)
	var found []wire.Entity
	for _, entity := range graph.Entities {
		if entity.Schema == want {
			found = append(found, entity)
		}
	}
	return found
}
func referenced(graph wire.Envelope, entity wire.Entity, field uint64) wire.Entity {
	return graph.Entities[value(entity, field).Reference]
}
func value(entity wire.Entity, field uint64) wire.Value {
	got, ok := entity.Fields[identity(field)]
	if !ok {
		fatal("missing required field")
	}
	return got
}
func identity(value uint64) wire.ID {
	id, err := wire.ParseID(fmt.Sprintf("%032x", value))
	check(err)
	return id
}
func check(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func fatal(message string) {
	fmt.Fprintln(os.Stderr, "structured-v10-check:", message)
	os.Exit(1)
}
