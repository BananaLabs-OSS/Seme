// Command structured-v11-check independently checks a generic v11 body graph.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 2 {
		fatal("usage: structured-v11-check PROGRAM.seme")
	}
	graph, err := wire.Read(os.Args[1])
	check(err)
	functions := bySchema(graph, 0x9011)
	blocks := bySchema(graph, 0x9080)
	returns := bySchema(graph, 0x9081)
	literals := bySchema(graph, 0x90b0)
	ands := bySchema(graph, 0x90b1)
	comparisons := bySchema(graph, 0x9021)
	booleanTypes := bySchema(graph, 0x9020)
	if len(functions) != 1 || len(blocks) != 1 || len(returns) != 1 || len(literals) != 1 || len(ands) != 1 || len(comparisons) != 1 || len(booleanTypes) != 1 {
		fatal("unexpected structured graph cardinality")
	}
	if referenced(graph, functions[0], 0x9112).Schema != identity(0x9020) {
		fatal("function result is not canonical Boolean type")
	}
	block := referenced(graph, functions[0], 0x9113)
	statements := value(block, 0x9800).List
	if block.Schema != identity(0x9080) || len(statements) != 1 {
		fatal("function body is not a one-statement Block")
	}
	returned := graph.Entities[statements[0].Reference]
	values := value(returned, 0x9810).List
	if returned.Schema != identity(0x9081) || len(values) != 1 || graph.Entities[values[0].Reference].Schema != identity(0x90b1) {
		fatal("Return root is not BooleanAnd")
	}
	and := graph.Entities[values[0].Reference]
	left := referenced(graph, and, 0x9b10)
	right := referenced(graph, and, 0x9b11)
	if left.Schema != identity(0x90b0) || value(left, 0x9b00).Tag != 2 || right.Schema != identity(0x9021) {
		fatal("ordered BooleanAnd operands were not preserved")
	}
	fmt.Println("Core Execution v11 short-circuit boolean graph passed")
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
	fmt.Fprintln(os.Stderr, "structured-v11-check:", message)
	os.Exit(1)
}
