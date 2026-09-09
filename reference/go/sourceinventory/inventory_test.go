package sourceinventory

import (
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/projectsource"
	"seme.local/reference/wire"
)

func contracts(t *testing.T) (contractcatalog.ProjectContractSet, contractcatalog.Contract) {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	execution, packages := read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme")
	v1, e := contractcatalog.ResolveProjectContractSet(execution, packages, read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	v2, e := contractcatalog.ResolveProjectContractSetV2(execution, packages, read("../../../modules/project/v2/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return v1, v2.Project()
}
func semanticProject(t *testing.T, set contractcatalog.ProjectContractSet) []byte {
	t.Helper()
	typ, param, body, fn, program := id("8001"), id("8002"), id("8003"), id("8004"), id("8005")
	execution := wire.Envelope{Module: id("9000"), Revision: id("8123"), Entities: map[wire.ID]wire.Entity{
		typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): unsigned(64), id("9101"): {Tag: 2}, id("9102"): unsigned(0)}}, param: {ID: param, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9120"): blob("v"), id("9121"): ref(typ), id("9122"): unsigned(0)}}, body: {ID: body, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(param)}}, fn: {ID: fn, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): blob("Apply"), id("9111"): list([]wire.Value{ref(param)}), id("9112"): ref(typ), id("9113"): ref(body)}}, program: {ID: program, Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{id("9150"): list([]wire.Value{ref(fn)}), id("9151"): ref(fn)}}}}
	out, e := projectemitter.Emit(set, projectemitter.Input{Identity: "example.test/inventory", RootPackage: "example.test/inventory", Execution: execution, Packages: []projectemitter.Package{{Name: "example.test/inventory", Interfaces: []projectemitter.Interface{{Name: "Apply", Function: fn, Parameters: []wire.ID{typ}, Result: typ}}}}})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func source(t *testing.T) projectsource.Snapshot {
	t.Helper()
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n// inventory bytes stay external\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "opaque.bin"), []byte("opaque external bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	out, e := projectsource.Discover(root, "example.test/inventory", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 8, MaxFileBytes: 1024, MaxTotalBytes: 4096})
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func TestDetachedInventoryRoundTripDeterministic(t *testing.T) {
	v1, v2 := contracts(t)
	project := semanticProject(t, v1)
	snapshot := source(t)
	a, e := Emit(v2, project, snapshot)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Emit(v2, project, snapshot)
	if e != nil {
		t.Fatal(e)
	}
	if string(a) != string(b) {
		t.Fatal("nondeterministic")
	}
	if e = Validate(v2, project, a); e != nil {
		t.Fatal(e)
	}
	for _, needle := range [][]byte{[]byte("package main"), []byte("inventory bytes stay external")} {
		if contains(a, needle) {
			t.Fatal("raw source bytes embedded")
		}
	}
}
func TestInventoryRejectsMutations(t *testing.T) {
	v1, v2 := contracts(t)
	project := semanticProject(t, v1)
	artifact, e := Emit(v2, project, source(t))
	if e != nil {
		t.Fatal(e)
	}
	tests := map[string]func(*wire.Envelope){"classification": func(e *wire.Envelope) {
		x, _ := one(*e, classificationSchema)
		n := e.Entities[x]
		n.Fields[id("e130")] = unsigned(5)
		e.Entities[x] = n
	}, "path": func(e *wire.Envelope) {
		x, _ := one(*e, unitSchema)
		n := e.Entities[x]
		n.Fields[id("e150")] = blob("../escape.go")
		e.Entities[x] = n
	}, "digest": func(e *wire.Envelope) {
		x, _ := one(*e, unitSchema)
		n := e.Entities[x]
		n.Fields[id("e151")] = bytesValue([]byte{1})
		e.Entities[x] = n
	}, "semantic-snapshot": func(e *wire.Envelope) {
		x, _ := one(*e, snapshotSchema)
		n := e.Entities[x]
		n.Version = 2
		e.Entities[x] = n
	}, "unknown-field": func(e *wire.Envelope) {
		x, _ := one(*e, inventorySchema)
		n := e.Entities[x]
		n.Fields[id("efff")] = wire.Value{}
		e.Entities[x] = n
	}, "unit-order": func(e *wire.Envelope) {
		x, _ := one(*e, inventorySchema)
		n := e.Entities[x]
		v := n.Fields[id("e164")]
		v.List[0], v.List[1] = v.List[1], v.List[0]
		n.Fields[id("e164")] = v
		e.Entities[x] = n
	}, "duplicate-unit": func(e *wire.Envelope) {
		x, _ := one(*e, inventorySchema)
		n := e.Entities[x]
		v := n.Fields[id("e164")]
		v.List[1] = v.List[0]
		n.Fields[id("e164")] = v
		e.Entities[x] = n
	}, "semantic-revision": func(e *wire.Envelope) {
		x, _ := one(*e, inventorySchema)
		n := e.Entities[x]
		v := n.Fields[id("e162")]
		v.Bytes[0] ^= 1
		n.Fields[id("e162")] = v
		e.Entities[x] = n
	}, "orphan-classification": func(e *wire.Envelope) {
		x := stable("classification", "99")
		e.Entities[x] = wire.Entity{ID: x, Schema: classificationSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e130"): unsigned(0)}}
	}, "orphan-unknown-schema": func(e *wire.Envelope) {
		x := stable("unknown", "entity")
		e.Entities[x] = wire.Entity{ID: x, Schema: id("efff"), Version: 1, Fields: map[wire.ID]wire.Value{}}
	}}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e, _ := wire.Decode(artifact)
			mutate(&e)
			e.Revision, _ = artifactRevision(e)
			bad, _ := wire.Encode(e)
			if Validate(v2, project, bad) == nil {
				t.Fatal("accepted mutation")
			}
		})
	}
}
func contains(haystack, needle []byte) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == string(needle) {
			return true
		}
	}
	return false
}
