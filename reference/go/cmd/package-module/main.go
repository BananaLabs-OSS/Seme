// Command package-module emits Package Contract v1 declarations.
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
		{0xb010, "Package", []field{f(0xb100, "package.name", 4, 0, 0), f(0xb101, "package.version", 4, 0, 0), f(0xb102, "package.interfaces", 5, 0xb011, 2), f(0xb103, "package.dependencies", 5, 0xb012, 2), f(0xb104, "package.required_effects", 5, 0, 2), f(0xb105, "package.runtime_assumptions", 5, 0xb013, 2), f(0xb106, "package.fidelity_mappings", 5, 0xb014, 2)}},
		{0xb011, "TypedInterface", []field{f(0xb110, "interface.name", 4, 0, 0), f(0xb111, "interface.function", 5, 0, 0), f(0xb112, "interface.parameter_types", 5, 0, 2), f(0xb113, "interface.result_type", 5, 0, 0)}},
		{0xb012, "Dependency", []field{f(0xb120, "dependency.name", 4, 0, 0), f(0xb121, "dependency.requirement", 4, 0, 0), f(0xb122, "dependency.resolution", 5, 0, 1)}},
		{0xb013, "RuntimeAssumption", []field{f(0xb130, "runtime_assumption.name", 4, 0, 0), f(0xb131, "runtime_assumption.detail", 4, 0, 0)}},
		{0xb014, "FidelityMapping", []field{f(0xb140, "fidelity.source", 5, 0, 0), f(0xb141, "fidelity.target", 5, 0, 0), f(0xb142, "fidelity.realization", 2, 0, 0), f(0xb143, "fidelity.evidence", 4, 0, 0)}},
	}
}
func main() {
	schemas := declarations()
	var fields []field
	for _, declaration := range schemas {
		fields = append(fields, declaration.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	fmt.Printf("# Generated construction projection for Package Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0xb000), id(0xb001), 1+len(schemas)+len(fields))
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(0xb000), id(0x12), id(0x120), text("package-contract-v1"), id(0x122), len(schemas)+len(fields))
	for _, declaration := range schemas {
		fmt.Printf("rf %s\n", id(declaration.id))
	}
	for _, field := range fields {
		fmt.Printf("rf %s\n", id(field.id))
	}
	for _, declaration := range schemas {
		emitSchema(declaration)
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
