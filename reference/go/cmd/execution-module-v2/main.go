// Command execution-module-v2 emits Core Execution Semantics revision 2.
package main

import (
	"encoding/hex"
	"fmt"
	"sort"
)

type field struct {
	id                 uint64
	name               string
	kind, schema, card uint64
}
type schema struct {
	id     uint64
	name   string
	fields []field
}

func id(n uint64) string       { return fmt.Sprintf("%032x", n) }
func text(value string) string { return hex.EncodeToString([]byte(value)) }
func f(id uint64, name string, kind, schema, card uint64) field {
	return field{id, name, kind, schema, card}
}
func declarations() []schema {
	return []schema{
		{0x9010, "IntegerType", []field{f(0x9100, "integer.width", 2, 0, 0), f(0x9101, "integer.signed", 1, 0, 0), f(0x9102, "integer.overflow", 2, 0, 0)}},
		{0x9011, "Function", []field{f(0x9110, "function.name", 4, 0, 0), f(0x9111, "function.parameters", 5, 0x9012, 2), f(0x9112, "function.result_type", 5, 0, 0), f(0x9113, "function.body", 5, 0, 0)}},
		{0x9012, "Parameter", []field{f(0x9120, "parameter.name", 4, 0, 0), f(0x9121, "parameter.type", 5, 0, 0), f(0x9122, "parameter.index", 2, 0, 0)}},
		{0x9013, "ParameterRead", []field{f(0x9130, "parameter_read.parameter", 5, 0x9012, 0)}},
		{0x9014, "IntegerAdd", []field{f(0x9140, "integer_add.left", 5, 0, 0), f(0x9141, "integer_add.right", 5, 0, 0), f(0x9142, "integer_add.type", 5, 0x9010, 0)}},
		{0x9015, "ExecutableProgram", []field{f(0x9150, "program.functions", 5, 0x9011, 2), f(0x9151, "program.entry", 5, 0x9011, 0)}},
		{0x9020, "BooleanType", nil},
		{0x9021, "IntegerLessEqual", []field{f(0x9160, "integer_less_equal.left", 5, 0, 0), f(0x9161, "integer_less_equal.right", 5, 0, 0), f(0x9162, "integer_less_equal.operand_type", 5, 0x9010, 0)}},
	}
}
func main() {
	schemas := declarations()
	var fields []field
	for _, schema := range schemas {
		fields = append(fields, schema.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	fmt.Printf("# Generated construction projection for Core Execution Semantics v2.\nve 1\nmo %s\nrv %s\npc 1\n%s\nec %d\n\n", id(0x9000), id(0x9002), id(0x9001), 1+len(schemas)+len(fields))
	fmt.Printf("en %s %s 2 2\nfi %s by %s\nfi %s li %d\n", id(0x9000), id(0x12), id(0x120), text("core-execution-v2"), id(0x122), len(schemas)+len(fields))
	for _, schema := range schemas {
		fmt.Printf("rf %s\n", id(schema.id))
	}
	for _, field := range fields {
		fmt.Printf("rf %s\n", id(field.id))
	}
	for _, schema := range schemas {
		emitSchema(schema)
	}
	for _, field := range fields {
		emitField(field)
	}
}
func emitSchema(schema schema) {
	fmt.Printf("\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(schema.id), id(0x10), id(0x100), text(schema.name), id(0x101), len(schema.fields))
	for _, field := range schema.fields {
		fmt.Printf("rf %s\n", id(field.id))
	}
}
func emitField(field field) {
	count := 1
	if field.schema != 0 {
		count = 2
	}
	fmt.Printf("\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", id(field.id), id(0x11), id(0x110), text(field.name), id(0x111), count, id(0x2000), field.kind)
	if field.schema != 0 {
		fmt.Printf("fi %s rf %s\n", id(0x2001), id(field.schema))
	}
	fmt.Printf("fi %s uu %d\nfi %s uu 1\n", id(0x112), field.card, id(0x113))
}
