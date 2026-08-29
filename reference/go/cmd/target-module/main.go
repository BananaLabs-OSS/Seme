// Command target-module emits Target Contract v1 declarations.
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
		{0xc010, "TargetContract", []field{f(0xc100, "target.name", 4, 0, 0), f(0xc101, "target.revision", 2, 0, 0), f(0xc102, "target.support_rules", 5, 0xc012, 2)}},
		{0xc011, "Requirement", []field{f(0xc110, "requirement.construct", 5, 0, 0), f(0xc111, "requirement.minimum_revision", 2, 0, 0), f(0xc112, "requirement.properties", 5, 0, 2)}},
		{0xc012, "SupportRule", []field{f(0xc120, "support.construct", 5, 0, 0), f(0xc121, "support.maximum_revision", 2, 0, 0), f(0xc122, "support.preserved_properties", 5, 0, 2), f(0xc123, "support.realization", 2, 0, 0), f(0xc124, "support.dependency", 5, 0, 1), f(0xc125, "support.evidence", 5, 0, 2)}},
		{0xc013, "Resolution", []field{f(0xc130, "resolution.requirement", 5, 0xc011, 0), f(0xc131, "resolution.target", 5, 0xc010, 0), f(0xc132, "resolution.selected_rule", 5, 0xc012, 1), f(0xc133, "resolution.realization", 2, 0, 0), f(0xc134, "resolution.unresolved_properties", 5, 0, 2), f(0xc135, "resolution.diagnostics", 5, 0, 2)}},
		{0xc014, "ExecutionPlan", []field{f(0xc140, "plan.root", 5, 0, 0), f(0xc141, "plan.target", 5, 0xc010, 0), f(0xc142, "plan.resolutions", 5, 0xc013, 2), f(0xc143, "plan.boundaries", 5, 0xc015, 2), f(0xc144, "plan.executable", 1, 0, 0)}},
		{0xc015, "Boundary", []field{f(0xc150, "boundary.provider", 5, 0, 0), f(0xc151, "boundary.consumer", 5, 0, 0), f(0xc152, "boundary.interface", 5, 0, 0), f(0xc153, "boundary.transport", 5, 0, 0)}},
	}
}
func main() {
	schemas := declarations()
	var fields []field
	for _, declaration := range schemas {
		fields = append(fields, declaration.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	fmt.Printf("# Generated construction projection for Target Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0xc000), id(0xc001), 1+len(schemas)+len(fields))
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(0xc000), id(0x12), id(0x120), text("target-contract-v1"), id(0x122), len(schemas)+len(fields))
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
