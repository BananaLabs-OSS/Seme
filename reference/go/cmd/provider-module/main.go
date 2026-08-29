// Command provider-module emits the checked G1 construction projection for
// Provider Contract v1. It is a differential generator, not authority.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"sort"
)

type field struct {
	id, name uint64
	text     string
	kind     uint64
	schema   uint64
	card     uint64
}
type schema struct {
	id, name uint64
	text     string
	fields   []field
}

func id(n uint64) string        { return fmt.Sprintf("%032x", n) }
func bytes(value string) string { return hex.EncodeToString([]byte(value)) }
func f(identity uint64, name string, kind, target, cardinality uint64) field {
	return field{identity, 0, name, kind, target, cardinality}
}

func declarations() []schema {
	return []schema{
		{0x7010, 0, "ProviderProfile", []field{
			f(0x7100, "provider.contract_version", 4, 0, 0),
			f(0x7101, "provider.identity", 4, 0, 0),
			f(0x7102, "provider.implementation_version", 4, 0, 0),
			f(0x7103, "provider.language", 4, 0, 0),
			f(0x7104, "provider.language_version", 4, 0, 0),
			f(0x7105, "provider.toolchain", 4, 0, 0),
			f(0x7106, "provider.target", 4, 0, 0),
			f(0x7107, "provider.environment", 4, 0, 0),
			f(0x7108, "provider.supported_constructs", 4, 0, 2),
			f(0x7109, "provider.exclusions", 4, 0, 2),
		}},
		{0x7011, 0, "NativeFile", []field{
			f(0x7110, "native_file.path", 4, 0, 0),
			f(0x7111, "native_file.digest", 4, 0, 0),
		}},
		{0x7012, 0, "SourceOccurrence", []field{
			f(0x7120, "occurrence.file", 5, 0x7011, 0),
			f(0x7121, "occurrence.start", 2, 0, 0),
			f(0x7122, "occurrence.end", 2, 0, 0),
			f(0x7123, "occurrence.role", 2, 0, 0),
		}},
		{0x7013, 0, "Declaration", []field{
			f(0x7130, "declaration.name", 4, 0, 0),
			f(0x7131, "declaration.qualified_name", 4, 0, 0),
			f(0x7132, "declaration.signature", 4, 0, 0),
			f(0x7133, "declaration.fidelity", 2, 0, 0),
			f(0x7134, "declaration.occurrences", 5, 0x7012, 2),
			f(0x7135, "declaration.identity_evidence", 5, 0x7014, 0),
		}},
		{0x7014, 0, "IdentityEvidence", []field{
			f(0x7140, "identity_evidence.native_key", 4, 0, 0),
			f(0x7141, "identity_evidence.semantic_fingerprint", 4, 0, 0),
			f(0x7142, "identity_evidence.matching_fingerprint", 4, 0, 0),
		}},
		{0x7015, 0, "OpaqueRegion", []field{
			f(0x7150, "opaque.file", 5, 0x7011, 0),
			f(0x7151, "opaque.start", 2, 0, 0),
			f(0x7152, "opaque.end", 2, 0, 0),
			f(0x7153, "opaque.digest", 4, 0, 0),
		}},
		{0x7016, 0, "IngestionResult", []field{
			f(0x7160, "ingestion.revision", 4, 0, 0),
			f(0x7161, "ingestion.profile", 5, 0x7010, 0),
			f(0x7162, "ingestion.files", 5, 0x7011, 2),
			f(0x7163, "ingestion.declarations", 5, 0x7013, 2),
			f(0x7164, "ingestion.opaque_regions", 5, 0x7015, 2),
		}},
		{0x7017, 0, "ProjectionReport", []field{
			f(0x7170, "projection.base_revision", 4, 0, 0),
			f(0x7171, "projection.result_revision", 4, 0, 0),
			f(0x7172, "projection.changed_files", 5, 0x7011, 2),
			f(0x7173, "projection.validation_command", 4, 0, 0),
			f(0x7174, "projection.validation_status", 2, 0, 0),
		}},
	}
}

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "provider-module: no arguments")
		os.Exit(64)
	}
	schemas := declarations()
	var fields []field
	for _, schema := range schemas {
		fields = append(fields, schema.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	fmt.Printf("# Generated construction projection for Provider Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", id(0x7000), id(0x7001), 1+len(schemas)+len(fields))
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(0x7000), id(0x12), id(0x120), bytes("provider-contract-v1"), id(0x122), len(schemas)+len(fields))
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
	fmt.Printf("\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(schema.id), id(0x10), id(0x100), bytes(schema.text), id(0x101), len(schema.fields))
	for _, field := range schema.fields {
		fmt.Printf("rf %s\n", id(field.id))
	}
}

func emitField(field field) {
	shapeFields := 1
	if field.schema != 0 {
		shapeFields = 2
	}
	fmt.Printf("\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", id(field.id), id(0x11), id(0x110), bytes(field.text), id(0x111), shapeFields, id(0x2000), field.kind)
	if field.schema != 0 {
		fmt.Printf("fi %s rf %s\n", id(0x2001), id(field.schema))
	}
	fmt.Printf("fi %s uu %d\nfi %s uu 1\n", id(0x112), field.card, id(0x113))
}
