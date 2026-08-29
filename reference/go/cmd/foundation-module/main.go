// Command foundation-module emits the checked G1 projection used to construct
// Semantic Foundation Module v1. It is a fixture generator, not an authority.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"seme.local/reference/foundation"
)

type field struct {
	id, name string
	kind     foundation.Kind
	schema   string
	card     foundation.Cardinality
}
type schema struct {
	id, name string
	fields   []field
}

func main() {
	schemas := declarations()
	var fields []field
	for _, schema := range schemas {
		fields = append(fields, schema.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	entityCount := 1 + len(schemas) + len(fields)
	fmt.Printf("# Generated construction projection for Semantic Foundation Module v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0x3000), id(0x3001), entityCount)
	for _, schema := range schemas {
		emitSchema(schema)
	}
	for _, field := range fields {
		emitField(field)
	}
	emitModule(schemas, fields)
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "foundation-module: no arguments")
		os.Exit(64)
	}
}

func declarations() []schema {
	return []schema{
		{id(0x10), "Schema", []field{f(0x100, "schema.name", foundation.Bytes, "", foundation.One), f(0x101, "schema.fields", foundation.Reference, id(0x11), foundation.Many), f(0x102, "schema.constraints", foundation.Reference, id(0x14), foundation.Many)}},
		{id(0x11), "Field", []field{f(0x110, "field.name", foundation.Bytes, "", foundation.One), f(0x111, "field.value_shape", foundation.Record, id(0x1c), foundation.One), f(0x112, "field.cardinality", foundation.Unsigned, "", foundation.One), f(0x113, "field.since_version", foundation.Unsigned, "", foundation.One)}},
		{id(0x12), "Module", []field{f(0x120, "module.name", foundation.Bytes, "", foundation.One), f(0x121, "module.imports", foundation.Reference, id(0x13), foundation.Many), f(0x122, "module.exports", foundation.Reference, "", foundation.Many), f(0x123, "module.required_effects", foundation.Reference, id(0x15), foundation.Many)}},
		{id(0x13), "Import", []field{f(0x130, "import.module", foundation.Reference, id(0x12), foundation.One), f(0x131, "import.required_revision", foundation.Bytes, "", foundation.One)}},
		{id(0x14), "Constraint", []field{f(0x140, "constraint.rule", foundation.Reference, id(0x19), foundation.One), f(0x141, "constraint.operation", foundation.Bytes, "", foundation.One)}},
		{id(0x15), "Effect", []field{f(0x150, "effect.name", foundation.Bytes, "", foundation.One), f(0x151, "effect.capability", foundation.Reference, id(0x16), foundation.One)}},
		{id(0x16), "Capability", []field{f(0x160, "capability.name", foundation.Bytes, "", foundation.One)}},
		{id(0x17), "Provenance", []field{f(0x170, "provenance.actor", foundation.Bytes, "", foundation.One), f(0x171, "provenance.parent_revision", foundation.Bytes, "", foundation.One)}},
		{id(0x18), "Refinement", []field{f(0x180, "refinement.from", foundation.Reference, "", foundation.One), f(0x181, "refinement.to", foundation.Reference, "", foundation.One), f(0x182, "refinement.evidence", foundation.Bytes, "", foundation.One)}},
		{id(0x19), "DiagnosticRule", []field{f(0x190, "diagnostic.name", foundation.Bytes, "", foundation.One), f(0x191, "diagnostic.arguments", foundation.Record, id(0x1c), foundation.Many)}},
		{id(0x1a), "Diagnostic", []field{f(0x1a0, "diagnostic.rule", foundation.Reference, id(0x19), foundation.One), f(0x1a1, "diagnostic.revision", foundation.Bytes, "", foundation.One), f(0x1a2, "diagnostic.entity", foundation.Bytes, "", foundation.One), f(0x1a3, "diagnostic.path", foundation.Record, id(0x1d), foundation.Many), f(0x1a4, "diagnostic.arguments", foundation.Any, "", foundation.Many), f(0x1a5, "diagnostic.severity", foundation.Unsigned, "", foundation.One)}},
		{id(0x1b), "DiagnosticReport", []field{f(0x1b0, "diagnostic_report.status", foundation.Unsigned, "", foundation.One), f(0x1b1, "diagnostic_report.diagnostics", foundation.Reference, id(0x1a), foundation.Many)}},
		{id(0x1c), "ValueShape", []field{f(0x2000, "value_shape.kind", foundation.Unsigned, "", foundation.One), f(0x2001, "value_shape.schema", foundation.Reference, id(0x10), foundation.Optional), f(0x2002, "value_shape.element", foundation.Record, id(0x1c), foundation.Optional)}},
		{id(0x1d), "PathSegment", []field{f(0x2100, "path_segment.kind", foundation.Unsigned, "", foundation.One), f(0x2101, "path_segment.identity", foundation.Bytes, "", foundation.Optional), f(0x2102, "path_segment.index", foundation.Unsigned, "", foundation.Optional)}},
		{id(0x1e), "ValidationResult", []field{f(0x2200, "validation_result.entity", foundation.Bytes, "", foundation.One), f(0x2201, "validation_result.disposition", foundation.Unsigned, "", foundation.One)}},
		{id(0x1f), "ValidationReport", []field{f(0x2210, "validation_report.revision", foundation.Bytes, "", foundation.One), f(0x2211, "validation_report.results", foundation.Record, id(0x1e), foundation.Many), f(0x2212, "validation_report.diagnostics", foundation.Reference, id(0x1a), foundation.Many)}},
	}
}

func emitModule(schemas []schema, fields []field) {
	exports := make([]string, 0, len(schemas)+len(fields))
	for _, s := range schemas {
		exports = append(exports, s.id)
	}
	for _, f := range fields {
		exports = append(exports, f.id)
	}
	sort.Strings(exports)
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(0x3000), id(0x12), id(0x120), text("semantic-foundation-v1"), id(0x122), len(exports))
	for _, export := range exports {
		fmt.Printf("rf %s\n", export)
	}
}
func emitSchema(s schema) {
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", s.id, id(0x10), id(0x100), text(s.name), id(0x101), len(s.fields))
	for _, f := range s.fields {
		fmt.Printf("rf %s\n", f.id)
	}
	fmt.Println()
}
func emitField(f field) {
	count := 1
	if f.schema != "" {
		count = 2
	}
	fmt.Printf("en %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", f.id, id(0x11), id(0x110), text(f.name), id(0x111), count, id(0x2000), f.kind)
	if f.schema != "" {
		fmt.Printf("fi %s rf %s\n", id(0x2001), f.schema)
	}
	fmt.Printf("fi %s uu %d\nfi %s uu 1\n\n", id(0x112), f.card, id(0x113))
}
func f(n uint64, name string, kind foundation.Kind, schema string, card foundation.Cardinality) field {
	return field{id(n), name, kind, schema, card}
}
func id(n uint64) string       { return fmt.Sprintf("%032x", n) }
func text(value string) string { return strings.ToLower(hex.EncodeToString([]byte(value))) }
