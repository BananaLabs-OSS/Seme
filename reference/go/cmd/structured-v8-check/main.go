// Command structured-v8-check independently checks the generic v8 body graph.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/wire"
)

func main() {
	if len(os.Args) != 2 {
		fatal("usage: structured-v8-check PROGRAM.seme")
	}
	graph, err := wire.Read(os.Args[1])
	check(err)
	functions := bySchema(graph, 0x9011)
	blocks := bySchema(graph, 0x9080)
	returns := bySchema(graph, 0x9081)
	adds := bySchema(graph, 0x9014)
	literals := bySchema(graph, 0x9070)
	if len(functions) != 1 || len(blocks) != 1 || len(returns) != 1 || len(adds) != 2 || len(literals) != 1 {
		fatal("unexpected structured graph cardinality")
	}
	block := referenced(graph, functions[0], 0x9113)
	if block.Schema != identity(0x9080) {
		fatal("function body is not a Block")
	}
	statements := value(block, 0x9800).List
	if len(statements) != 1 || statements[0].Reference != returns[0].ID {
		fatal("block does not contain the Return")
	}
	values := value(returns[0], 0x9810).List
	if len(values) != 1 || graph.Entities[values[0].Reference].Schema != identity(0x9014) {
		fatal("Return does not contain the compositional addition")
	}
	if value(literals[0], 0x9700).Unsigned != 1 {
		fatal("integer literal meaning was not preserved")
	}
	fmt.Println("Core Execution v8 structured graph passed")
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
	fmt.Fprintln(os.Stderr, "structured-v8-check:", message)
	os.Exit(1)
}
