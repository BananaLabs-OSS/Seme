// Command package-module emits Package Contract v1 declarations.
package main

import (
	"encoding/hex"
	"flag"
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
func declarations(version int) []schema {
	out := []schema{
		{0xb010, "Package", []field{f(0xb100, "package.name", 4, 0, 0), f(0xb101, "package.version", 4, 0, 0), f(0xb102, "package.interfaces", 5, 0xb011, 2), f(0xb103, "package.dependencies", 5, 0xb012, 2), f(0xb104, "package.required_effects", 5, 0, 2), f(0xb105, "package.runtime_assumptions", 5, 0xb013, 2), f(0xb106, "package.fidelity_mappings", 5, 0xb014, 2)}},
		{0xb011, "TypedInterface", []field{f(0xb110, "interface.name", 4, 0, 0), f(0xb111, "interface.function", 5, 0, 0), f(0xb112, "interface.parameter_types", 5, 0, 2), f(0xb113, "interface.result_type", 5, 0, 0)}},
		{0xb012, "Dependency", []field{f(0xb120, "dependency.name", 4, 0, 0), f(0xb121, "dependency.requirement", 4, 0, 0), f(0xb122, "dependency.resolution", 5, 0, 1)}},
		{0xb013, "RuntimeAssumption", []field{f(0xb130, "runtime_assumption.name", 4, 0, 0), f(0xb131, "runtime_assumption.detail", 4, 0, 0)}},
		{0xb014, "FidelityMapping", []field{f(0xb140, "fidelity.source", 5, 0, 0), f(0xb141, "fidelity.target", 5, 0, 0), f(0xb142, "fidelity.realization", 2, 0, 0), f(0xb143, "fidelity.evidence", 4, 0, 0)}},
	}
	if version >= 2 {
		out = append(out,
			schema{0xb020, "PackageGraph", []field{f(0xb200, "package_graph.packages", 5, 0xb021, 2), f(0xb201, "package_graph.content_revision", 4, 0, 0)}},
			schema{0xb021, "PackageDetail", []field{f(0xb210, "package_detail.package", 5, 0xb010, 0), f(0xb211, "package_detail.members", 5, 0xb022, 2), f(0xb212, "package_detail.imports", 5, 0xb024, 2), f(0xb213, "package_detail.origins", 5, 0xb026, 2), f(0xb214, "package_detail.revision", 4, 0, 0)}},
			schema{0xb022, "DeclarationMember", []field{f(0xb220, "declaration_member.declaration", 5, 0, 0), f(0xb221, "declaration_member.name", 4, 0, 0), f(0xb222, "declaration_member.visibility", 5, 0xb023, 0), f(0xb223, "declaration_member.export_name", 4, 0, 1), f(0xb224, "declaration_member.origin", 5, 0xb026, 0)}},
			schema{0xb023, "Visibility", []field{f(0xb230, "visibility.code", 2, 0, 0)}},
			schema{0xb024, "ImportBinding", []field{f(0xb240, "import_binding.alias", 4, 0, 1), f(0xb241, "import_binding.requested_identity", 4, 0, 0), f(0xb242, "import_binding.classification", 5, 0xb025, 0), f(0xb243, "import_binding.resolved_local_package", 5, 0xb010, 1), f(0xb244, "import_binding.external_resolution", 5, 0xb012, 1), f(0xb245, "import_binding.origin", 5, 0xb026, 0)}},
			schema{0xb025, "ImportClass", []field{f(0xb250, "import_class.code", 2, 0, 0)}},
			schema{0xb026, "SourceOrigin", []field{f(0xb260, "source_origin.source_unit_identity", 5, 0, 0), f(0xb261, "source_origin.normalized_relative_path", 4, 0, 0), f(0xb262, "source_origin.content_digest", 4, 0, 0), f(0xb263, "source_origin.start_byte", 2, 0, 0), f(0xb264, "source_origin.end_byte", 2, 0, 0), f(0xb265, "source_origin.start_line", 2, 0, 0), f(0xb266, "source_origin.start_column", 2, 0, 0), f(0xb267, "source_origin.end_line", 2, 0, 0), f(0xb268, "source_origin.end_column", 2, 0, 0)}},
		)
	}
	if version == 3 {
		out = append(out,
			schema{0xb027, "SemanticDeclarationKind", []field{f(0xb270, "semantic_declaration_kind.code", 2, 0, 0)}},
			schema{0xb028, "OwnedSemanticDeclaration", []field{
				f(0xb280, "owned_semantic_declaration.declaration", 5, 0, 0),
				f(0xb281, "owned_semantic_declaration.owner", 5, 0xb021, 0),
				f(0xb282, "owned_semantic_declaration.kind", 5, 0xb027, 0),
				f(0xb283, "owned_semantic_declaration.name", 4, 0, 0),
				f(0xb284, "owned_semantic_declaration.visibility", 5, 0xb023, 0),
				f(0xb285, "owned_semantic_declaration.export_name", 4, 0, 1),
				f(0xb286, "owned_semantic_declaration.origin", 5, 0xb026, 0),
				f(0xb287, "owned_semantic_declaration.referenced_imports", 5, 0xb024, 2),
				f(0xb288, "owned_semantic_declaration.generic_definition", 5, 0, 1),
			}},
			schema{0xb029, "CompletePackageGraph", []field{
				f(0xb290, "complete_package_graph.package_graph", 5, 0xb020, 0),
				f(0xb291, "complete_package_graph.supplemental_declarations", 5, 0xb028, 2),
				f(0xb292, "complete_package_graph.content_revision", 4, 0, 0),
			}},
		)
	}
	return out
}
func main() {
	version := flag.Int("version", 1, "Package Contract version (1, 2, or 3)")
	flag.Parse()
	if *version < 1 || *version > 3 {
		panic("unsupported Package Contract version")
	}
	schemas := declarations(*version)
	var fields []field
	for _, declaration := range schemas {
		fields = append(fields, declaration.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	parents := 0
	if *version >= 2 {
		parents = 1
	}
	fmt.Printf("# Generated construction projection for Package Contract v%d.\nve 1\nmo %s\nrv %s\npc %d\n", *version, id(0xb000), id(0xb000+uint64(*version)), parents)
	if parents == 1 {
		fmt.Printf("%s\n", id(0xb000+uint64(*version)-1))
	}
	fmt.Printf("ec %d\n\n", 1+len(schemas)+len(fields))
	fmt.Printf("en %s %s %d 2\nfi %s by %s\nfi %s li %d\n", id(0xb000), id(0x12), *version, id(0x120), text(fmt.Sprintf("package-contract-v%d", *version)), id(0x122), len(schemas)+len(fields))
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
