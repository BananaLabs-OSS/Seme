package dependencyemitter_test

import (
	"bytes"
	"os"
	"reflect"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/wire"
)

func contract(t *testing.T) contractcatalog.Contract {
	t.Helper()
	b, err := os.ReadFile("../../../modules/dependency/v1/module.seme")
	if err != nil {
		t.Fatal(err)
	}
	c, err := contractcatalog.ResolveDependencyContract(b)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func closure() dependencyresolution.Closure {
	return dependencyresolution.Closure{
		Requirements: []dependencyresolution.Requirement{
			{Identity: "example.test/local", Requirement: "source:abc", Kind: dependencyresolution.Local, Metadata: []dependencyresolution.Metadata{{Key: "direct", Value: "true"}}},
			{Identity: "example.test/remote", Requirement: "v1.2.3", Kind: dependencyresolution.External},
		},
		Entries: []dependencyresolution.Entry{
			{Identity: "example.test/local", Version: "source:abc", Integrity: "h1:local", IntegrityAlgorithm: "go-h1-tree", Source: "local", SourceKind: "tree", Digest: "abc", Kind: dependencyresolution.Local, Dependencies: []string{"example.test/remote"}},
			{Identity: "example.test/remote", Ecosystem: "go", Version: "v1.2.3", Integrity: "h1:remote", IntegrityAlgorithm: "go-h1-tree", Source: "cache", SourceKind: "module-tree", Digest: "def", Kind: dependencyresolution.External, Metadata: []dependencyresolution.Metadata{{Key: "direct", Value: "true"}}},
		},
	}
}

func TestEmitRepeatValidateAndTamper(t *testing.T) {
	c := contract(t)
	in := closure()
	want := closure()
	a, err := dependencyemitter.Emit(c, in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := dependencyemitter.Emit(c, in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("nondeterministic")
	}
	if !reflect.DeepEqual(in, want) {
		t.Fatal("input mutated")
	}
	got, err := dependencyinstance.Validate(c, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 2 || len(got.Requirements) != 2 {
		t.Fatalf("bad roundtrip: %#v", got)
	}
	e, err := wire.Decode(a)
	if err != nil {
		t.Fatal(err)
	}
	for id, q := range e.Entities {
		if q.Schema.String() == "0000000000000000000000000000f010" {
			q.Fields[mustID(t, "0000000000000000000000000000f102")] = wire.Value{Tag: 5, Bytes: bytes.Repeat([]byte{9}, 32)}
			e.Entities[id] = q
		}
	}
	tampered, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dependencyinstance.Validate(c, tampered); err == nil {
		t.Fatal("accepted tamper")
	}
}

func TestRejectsInvalidWithoutOutput(t *testing.T) {
	c := contract(t)
	tests := []dependencyresolution.Closure{closure(), closure(), closure()}
	tests[0].Entries[0].Dependencies = []string{"example.test/local"}
	tests[1].Entries = append(tests[1].Entries, tests[1].Entries[1])
	tests[2].Entries[0], tests[2].Entries[1] = tests[2].Entries[1], tests[2].Entries[0]
	for i, x := range tests {
		if out, err := dependencyemitter.Emit(c, x); err == nil || out != nil {
			t.Fatalf("case %d: output=%x err=%v", i, out, err)
		}
	}
	if out, err := dependencyemitter.Emit(contractcatalog.Contract{}, closure()); err == nil || out != nil {
		t.Fatalf("wrong contract accepted")
	}
}

func TestValidatorRejectsOrphanUnsortedAndWrongContract(t *testing.T) {
	c := contract(t)
	b, err := dependencyemitter.Emit(c, closure())
	if err != nil {
		t.Fatal(err)
	}
	base, err := wire.Decode(b)
	if err != nil {
		t.Fatal(err)
	}

	orphan := base
	orphan.Entities = cloneEntities(base.Entities)
	oid := mustID(t, "11111111111111111111111111111111")
	orphan.Entities[oid] = wire.Entity{ID: oid, Schema: mustID(t, "0000000000000000000000000000f015"), Version: 1, Fields: map[wire.ID]wire.Value{
		mustID(t, "0000000000000000000000000000f150"): {Tag: 5, Bytes: []byte("x")}, mustID(t, "0000000000000000000000000000f151"): {Tag: 5, Bytes: []byte("y")}}}
	ob, _ := wire.Encode(orphan)
	if _, err = dependencyinstance.Validate(c, ob); err == nil {
		t.Fatal("accepted orphan")
	}

	unsorted := base
	unsorted.Entities = cloneEntities(base.Entities)
	for id, q := range unsorted.Entities {
		if q.Schema.String() == "0000000000000000000000000000f010" {
			f := mustID(t, "0000000000000000000000000000f101")
			v := q.Fields[f]
			v.List[0], v.List[1] = v.List[1], v.List[0]
			q.Fields[f] = v
			unsorted.Entities[id] = q
		}
	}
	ub, _ := wire.Encode(unsorted)
	if _, err = dependencyinstance.Validate(c, ub); err == nil {
		t.Fatal("accepted unsorted references")
	}
	if _, err = dependencyinstance.Validate(contractcatalog.Contract{}, b); err == nil {
		t.Fatal("accepted untrusted contract")
	}
}

func cloneEntities(in map[wire.ID]wire.Entity) map[wire.ID]wire.Entity {
	out := make(map[wire.ID]wire.Entity, len(in))
	for id, q := range in {
		fields := make(map[wire.ID]wire.Value, len(q.Fields))
		for k, v := range q.Fields {
			fields[k] = v
		}
		q.Fields = fields
		out[id] = q
	}
	return out
}

func mustID(t *testing.T, s string) wire.ID {
	t.Helper()
	x, err := wire.ParseID(s)
	if err != nil {
		t.Fatal(err)
	}
	return x
}
