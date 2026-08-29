// Command patch-module emits the checked G1 projection used to construct Patch
// Module v1. It is a fixture generator, not canonical authority.
package main

import (
	"encoding/hex"
	"fmt"
	"os"
)

type field struct {
	id, name uint64
	text     string
	kind     uint64
	schema   uint64
	card     uint64
}

func id(n uint64) string        { return fmt.Sprintf("%032x", n) }
func bytes(value string) string { return hex.EncodeToString([]byte(value)) }

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "patch-module: no arguments")
		os.Exit(64)
	}
	patchFields := []field{
		{0x5100, 0, "patch.author", 5, 0, 0},
		{0x5101, 0, "patch.base_revision", 4, 0, 0},
		{0x5102, 0, "patch.operations", 5, 0x5011, 2},
	}
	renameFields := []field{
		{0x5110, 0, "rename.target", 4, 0, 0},
		{0x5111, 0, "rename.field", 4, 0, 0},
		{0x5112, 0, "rename.expected_value", 4, 0, 0},
		{0x5113, 0, "rename.replacement", 4, 0, 0},
	}
	fmt.Printf("# Generated construction projection for Patch Module v1.\nve 1\nmo %s\nrv %s\npc 0\nec 10\n\n", id(0x5000), id(0x5001))
	fmt.Printf("en %s %s 1 2\nfi %s by %s\nfi %s li 9\n", id(0x5000), id(0x12), id(0x120), bytes("patch-v1"), id(0x122))
	for _, n := range []uint64{0x5010, 0x5011, 0x5100, 0x5101, 0x5102, 0x5110, 0x5111, 0x5112, 0x5113} {
		fmt.Printf("rf %s\n", id(n))
	}
	emitSchema(0x5010, "Patch", patchFields)
	emitSchema(0x5011, "RenameDeclaration", renameFields)
	for _, f := range append(patchFields, renameFields...) {
		emitField(f)
	}
}

func emitSchema(identity uint64, name string, fields []field) {
	fmt.Printf("\nen %s %s 1 2\nfi %s by %s\nfi %s li %d\n", id(identity), id(0x10), id(0x100), bytes(name), id(0x101), len(fields))
	for _, f := range fields {
		fmt.Printf("rf %s\n", id(f.id))
	}
}

func emitField(f field) {
	count := 1
	if f.schema != 0 {
		count = 2
	}
	fmt.Printf("\nen %s %s 1 4\nfi %s by %s\nfi %s rc %d\nfi %s uu %d\n", id(f.id), id(0x11), id(0x110), bytes(f.text), id(0x111), count, id(0x2000), f.kind)
	if f.schema != 0 {
		fmt.Printf("fi %s rf %s\n", id(0x2001), id(f.schema))
	}
	fmt.Printf("fi %s uu %d\nfi %s uu 1\n", id(0x112), f.card, id(0x113))
}
