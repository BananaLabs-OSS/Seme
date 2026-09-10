// Package sourcepresentationmodule emits Source Presentation Contract v1.
package sourcepresentationmodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000001000"
const RevisionID = "00000000000000000000000000001001"

type Visibility uint64

const (
	PackageVisibility Visibility = iota
	PublicVisibility
)

func ValidateVisibility(v uint64) error {
	if v > uint64(PublicVisibility) {
		return fmt.Errorf("source presentation visibility %d outside closed range 0..1", v)
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

func f(id uint64, name string, kind, target, card uint64) field {
	return field{id, name, kind, target, card}
}
func declarations() []schema {
	return []schema{
		{0x1010, "PresentationManifest", []field{f(0x1100, "presentation_manifest.project_snapshot", 5, 0xe024, 0), f(0x1101, "presentation_manifest.aliases", 5, 0x1011, 2), f(0x1102, "presentation_manifest.content_revision", 4, 0, 0)}},
		{0x1011, "TypeAliasPresentation", []field{f(0x1110, "type_alias.owner", 5, 0xb010, 0), f(0x1111, "type_alias.stable_name", 4, 0, 0), f(0x1112, "type_alias.visibility", 5, 0x1012, 0), f(0x1113, "type_alias.target_type", 5, 0, 0), f(0x1114, "type_alias.source_unit", 5, 0xe015, 0), f(0x1115, "type_alias.origin", 5, 0xb026, 0), f(0x1116, "type_alias.referenced_import_bindings", 5, 0xb024, 2)}},
		{0x1012, "PresentationVisibility", []field{f(0x1120, "presentation_visibility.code", 2, 0, 0)}},
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
	exports := len(d) + len(fields)
	fmt.Fprintf(out, "# Generated construction projection for Source Presentation Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 4+exports)
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 3\nrf %s\nrf %s\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("source-presentation-contract-v1"), id(0x121), id(0x1002), id(0x1003), id(0x1004), id(0x122), exports)
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	for _, p := range []struct{ id, module, rev uint64 }{{0x1002, 0xb000, 0xb004}, {0x1003, 0x9000, 0x9024}, {0x1004, 0xe000, 0xe00b}} {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(p.id), id(0x13), id(0x130), id(p.module), id(0x131), id(p.rev))
	}
	for _, s := range d {
		fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(s.id), id(0x10), id(0x100), text(s.name), id(0x101), len(s.fields))
		for _, x := range s.fields {
			fmt.Fprintf(out, "rf %s\n", id(x.id))
		}
	}
	for _, x := range fields {
		count, constraint := 1, ""
		if x.schema != 0 {
			count = 2
			constraint = fmt.Sprintf("fi %s rf %s\n", id(0x2001), id(x.schema))
		}
		fmt.Fprintf(out, "\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n%sfi %s uu %d\nfi %s uu 1\n", id(x.id), id(0x11), id(0x110), text(x.name), id(0x111), count, id(0x2000), x.kind, constraint, id(0x112), x.card, id(0x113))
	}
	return nil
}
