// Package dependencymodule emits the language-neutral Dependency Contract v1.
package dependencymodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "0000000000000000000000000000f000"
const RevisionID = "0000000000000000000000000000f001"

type Kind uint64

const (
	Local Kind = iota
	Ecosystem
)

func ValidateKind(x uint64) error {
	if x > uint64(Ecosystem) {
		return fmt.Errorf("dependency kind %d outside closed range 0..1", x)
	}
	return nil
}

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

func f(id uint64, name string, kind, schema, card uint64) field {
	return field{id, name, kind, schema, card}
}
func declarations() []schema {
	return []schema{
		{0xf010, "DependencyClosure", []field{f(0xf100, "dependency_closure.root_requirements", 5, 0xf011, 2), f(0xf101, "dependency_closure.resolutions", 5, 0xf012, 2), f(0xf102, "dependency_closure.content_revision", 4, 0, 0)}},
		{0xf011, "RootRequirement", []field{f(0xf110, "root_requirement.identity", 4, 0, 0), f(0xf111, "root_requirement.requirement", 4, 0, 0), f(0xf112, "root_requirement.kind", 5, 0xf014, 0), f(0xf113, "root_requirement.metadata", 5, 0xf015, 2)}},
		{0xf012, "ResolvedDependency", []field{f(0xf120, "resolved_dependency.identity", 4, 0, 0), f(0xf121, "resolved_dependency.ecosystem", 5, 0xf013, 1), f(0xf122, "resolved_dependency.version", 4, 0, 0), f(0xf123, "resolved_dependency.integrity", 5, 0xf016, 0), f(0xf124, "resolved_dependency.source", 5, 0xf017, 0), f(0xf125, "resolved_dependency.kind", 5, 0xf014, 0), f(0xf126, "resolved_dependency.dependencies", 5, 0xf012, 2), f(0xf127, "resolved_dependency.metadata", 5, 0xf015, 2)}},
		{0xf013, "Ecosystem", []field{f(0xf130, "ecosystem.identity", 4, 0, 0)}},
		{0xf014, "DependencyKind", []field{f(0xf140, "dependency_kind.code", 2, 0, 0)}},
		{0xf015, "DependencyMetadata", []field{f(0xf150, "dependency_metadata.key", 4, 0, 0), f(0xf151, "dependency_metadata.value", 4, 0, 0)}},
		{0xf016, "DependencyIntegrity", []field{f(0xf160, "dependency_integrity.algorithm", 4, 0, 0), f(0xf161, "dependency_integrity.digest", 4, 0, 0)}},
		{0xf017, "DependencySource", []field{f(0xf170, "dependency_source.kind", 4, 0, 0), f(0xf171, "dependency_source.identity", 4, 0, 0), f(0xf172, "dependency_source.metadata", 5, 0xf015, 2)}},
	}
}
func Emit(out io.Writer) error {
	d := declarations()
	fields := []field{}
	for _, s := range d {
		fields = append(fields, s.fields...)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].id < fields[j].id })
	id := func(x uint64) string { return fmt.Sprintf("%032x", x) }
	text := func(x string) string { return hex.EncodeToString([]byte(x)) }
	fmt.Fprintf(out, "# Generated construction projection for Dependency Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 1+len(d)+len(fields))
	fmt.Fprintf(out, "en %s %s 1 2\nfi %s by %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("dependency-contract-v1"), id(0x122), len(d)+len(fields))
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, s := range d {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(s.id), id(0x10), id(0x100), text(s.name), id(0x101), len(s.fields))
		for _, x := range s.fields {
			fmt.Fprintf(out, "rf %s\n", id(x.id))
		}
	}
	for _, x := range fields {
		count := 1
		constraint := ""
		if x.schema != 0 {
			count = 2
			constraint = fmt.Sprintf("fi %s rf %s\n", id(0x2001), id(x.schema))
		}
		fmt.Fprintf(out, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n%sfi %s uu %d\nfi %s uu 1\n", id(x.id), id(0x11), id(0x110), text(x.name), id(0x111), count, id(0x2000), x.kind, constraint, id(0x112), x.card, id(0x113))
	}
	return nil
}
