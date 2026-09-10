// Package resourcemodule emits the language-neutral Resource Contract v1.
package resourcemodule

import (
	"encoding/hex"
	"fmt"
	"io"
	"sort"
)

const ModuleID = "00000000000000000000000000006000"
const RevisionID = "00000000000000000000000000006001"

type RepresentationKind uint64

const (
	Text RepresentationKind = iota
	Bytes
)

func ValidateRepresentationKind(v uint64) error {
	if v > uint64(Bytes) {
		return fmt.Errorf("resource representation kind %d outside closed range 0..1", v)
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
		{0x6010, "ResourceManifest", []field{f(0x6100, "resource_manifest.resources", 5, 0x6011, 2), f(0x6101, "resource_manifest.placements", 5, 0x6013, 2), f(0x6102, "resource_manifest.content_revision", 4, 0, 0)}},
		{0x6011, "Resource", []field{f(0x6110, "resource.logical_identity", 4, 0, 0), f(0x6111, "resource.owner", 5, 0xb010, 0), f(0x6112, "resource.source_unit", 5, 0, 0), f(0x6113, "resource.normalized_path", 4, 0, 0), f(0x6114, "resource.representation_kind", 5, 0x6012, 0), f(0x6115, "resource.media_type", 4, 0, 0), f(0x6116, "resource.byte_size", 2, 0, 0), f(0x6117, "resource.sha256", 4, 0, 0)}},
		{0x6012, "RepresentationKind", []field{f(0x6120, "representation_kind.code", 2, 0, 0)}},
		{0x6013, "Placement", []field{f(0x6130, "placement.resource", 5, 0x6011, 0), f(0x6131, "placement.destination", 4, 0, 0)}},
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
	fmt.Fprintf(out, "# Generated construction projection for Resource Contract v1.\nve 1\nmo %s\nrv %s\npc 0\nec %d\n\n", ModuleID, RevisionID, 2+len(d)+len(fields))
	fmt.Fprintf(out, "en %s %s 1 3\nfi %s by %s\nfi %s li 1\nrf %s\nfi %s li %d\n", ModuleID, id(0x12), id(0x120), text("resource-contract-v1"), id(0x121), id(0x6002), id(0x122), len(d)+len(fields))
	for _, s := range d {
		fmt.Fprintf(out, "rf %s\n", id(s.id))
	}
	for _, x := range fields {
		fmt.Fprintf(out, "rf %s\n", id(x.id))
	}
	fmt.Fprintf(out, "\nen %s %s 1 2\nfi %s rf %s\nfi %s by %s\n", id(0x6002), id(0x13), id(0x130), id(0xb000), id(0x131), id(0xb004))
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
