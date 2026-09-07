// Package languageservice defines the provider-neutral live language-service
// contract and its reference state transition.
package languageservice

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

type Field struct {
	ID                 uint64
	Name               string
	Kind, Schema, Card uint64
}

type Schema struct {
	ID     uint64
	Name   string
	Fields []Field
}

func f(id uint64, name string, kind, schema, card uint64) Field {
	return Field{id, name, kind, schema, card}
}

func Declarations() []Schema {
	return []Schema{
		{0xd010, "DocumentIdentity", []Field{f(0xd100, "document.key", 4, 0, 0)}},
		{0xd011, "UpdateRequest", []Field{
			f(0xd110, "update.document", 5, 0xd010, 0), f(0xd111, "update.client_revision", 2, 0, 0),
			f(0xd112, "update.content_digest", 4, 0, 0), f(0xd113, "update.content", 4, 0, 0),
		}},
		{0xd012, "LanguageDiagnostic", []Field{
			f(0xd120, "diagnostic.severity", 2, 0, 0), f(0xd121, "diagnostic.code", 4, 0, 0),
			f(0xd122, "diagnostic.message", 4, 0, 0), f(0xd123, "diagnostic.start", 2, 0, 1),
			f(0xd124, "diagnostic.end", 2, 0, 1),
		}},
		{0xd013, "SemanticSourceMapping", []Field{
			f(0xd130, "mapping.semantic_identity", 4, 0, 0), f(0xd131, "mapping.document", 5, 0xd010, 0),
			f(0xd132, "mapping.start", 2, 0, 0), f(0xd133, "mapping.end", 2, 0, 0), f(0xd134, "mapping.role", 2, 0, 0),
		}},
		{0xd014, "LiftResult", []Field{
			f(0xd140, "lift.document", 5, 0xd010, 0), f(0xd141, "lift.client_revision", 2, 0, 0),
			f(0xd142, "lift.content_digest", 4, 0, 0), f(0xd143, "lift.disposition", 2, 0, 0),
			f(0xd144, "lift.canonical_revision", 4, 0, 1), f(0xd145, "lift.last_valid_revision", 4, 0, 1),
			f(0xd146, "lift.diagnostics", 5, 0xd012, 2), f(0xd147, "lift.source_mappings", 5, 0xd013, 2),
		}},
		{0xd015, "DocumentState", []Field{
			f(0xd150, "state.document", 5, 0xd010, 0), f(0xd151, "state.client_revision", 2, 0, 0),
			f(0xd152, "state.content_digest", 4, 0, 0), f(0xd153, "state.last_valid_revision", 4, 0, 1),
		}},
	}
}

func Emit(out io.Writer) {
	schemas := Declarations()
	var fields []Field
	for _, schema := range schemas {
		fields = append(fields, schema.Fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].ID < fields[j].ID })
	fmt.Fprintf(out, "# Generated construction projection for Live Language Service Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0xd000), id(0xd001), 1+len(schemas)+len(fields))
	fmt.Fprintf(out, "en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(0xd000), id(0x12), id(0x120), text("live-language-service-v1"), id(0x122), len(schemas)+len(fields))
	for _, schema := range schemas {
		fmt.Fprintf(out, "rf %s\n", id(schema.ID))
	}
	for _, field := range fields {
		fmt.Fprintf(out, "rf %s\n", id(field.ID))
	}
	for _, schema := range schemas {
		emitSchema(out, schema)
	}
	for _, field := range fields {
		emitField(out, field)
	}
}

func emitSchema(out io.Writer, schema Schema) {
	fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(schema.ID), id(0x10), id(0x100), text(schema.Name), id(0x101), len(schema.Fields))
	for _, field := range schema.Fields {
		fmt.Fprintf(out, "rf %s\n", id(field.ID))
	}
}

func emitField(out io.Writer, field Field) {
	count := 1
	if field.Schema != 0 {
		count = 2
	}
	fmt.Fprintf(out, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", id(field.ID), id(0x11), id(0x110), text(field.Name), id(0x111), count, id(0x2000), field.Kind)
	if field.Schema != 0 {
		fmt.Fprintf(out, "fi %s rf %s\n", id(0x2001), id(field.Schema))
	}
	fmt.Fprintf(out, "fi %s uu %d\nfi %s uu 1\n", id(0x112), field.Card, id(0x113))
}

func id(value uint64) string   { return fmt.Sprintf("%032x", value) }
func text(value string) string { return hex.EncodeToString([]byte(value)) }
