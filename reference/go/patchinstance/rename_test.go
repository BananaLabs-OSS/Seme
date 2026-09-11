package patchinstance

import (
	"bytes"
	"os"
	"testing"

	"seme.local/reference/canonicalclosure"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func TestEmitRenameIsCanonicalDeterministicAndBoundToBase(t *testing.T) {
	source, err := os.ReadFile("../../../modules/patch/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := contractcatalog.ResolvePatchContract(source)
	if err != nil {
		t.Fatal(err)
	}
	target, field := id("8001"), id("8002")
	base := wire.Envelope{Module: id("8000"), Revision: id("8003"), Entities: map[wire.ID]wire.Entity{
		target: entity(target, "8004", map[string]wire.Value{"8002": blob([]byte("Before"))}),
	}}
	in := Rename{Base: base, Target: target, Field: field, Expected: []byte("Before"), Replacement: []byte("After")}
	a, err := EmitRename(contract, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EmitRename(contract, in)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("rename instance is nondeterministic")
	}
	graph, err := wire.Decode(a)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := wire.Encode(graph)
	if err != nil || !bytes.Equal(a, canonical) {
		t.Fatal("rename instance is not canonical")
	}
	if err = canonicalclosure.Validate(a); err != nil {
		t.Fatalf("patch closure: %v", err)
	}
	count := 0
	for _, candidate := range graph.Entities {
		if candidate.Schema == id("5010") {
			count++
			if !bytes.Equal(candidate.Fields[id("5101")].Bytes, base.Revision[:]) {
				t.Fatal("base revision not bound")
			}
		}
	}
	if count != 1 {
		t.Fatalf("patch count=%d", count)
	}
}

func TestEmitRenameRejectsWrongPreconditionWithoutArtifact(t *testing.T) {
	source, _ := os.ReadFile("../../../modules/patch/v1/module.seme")
	contract, _ := contractcatalog.ResolvePatchContract(source)
	target, field := id("8101"), id("8102")
	base := wire.Envelope{Module: id("8100"), Revision: id("8103"), Entities: map[wire.ID]wire.Entity{
		target: entity(target, "8104", map[string]wire.Value{"8102": blob([]byte("Before"))}),
	}}
	artifact, err := EmitRename(contract, Rename{Base: base, Target: target, Field: field, Expected: []byte("Wrong"), Replacement: []byte("After")})
	if err == nil || artifact != nil {
		t.Fatal("wrong precondition published")
	}
}
