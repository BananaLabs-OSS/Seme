package gopackageadapter

import (
	"os"
	"path/filepath"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprovider"
	"seme.local/reference/projectemitter"
	"seme.local/reference/projectsource"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

func TestConsumedImportRealizationsAreClosedAndConstructBacked(t *testing.T) {
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		testID("100"): {ID: testID("100"), Schema: testID("a065"), Version: 1},
		testID("101"): {ID: testID("101"), Schema: testID("a043"), Version: 1},
		testID("102"): {ID: testID("102"), Schema: testID("90fc"), Version: 1},
		testID("103"): {ID: testID("103"), Schema: testID("15"), Version: 1, Fields: map[wire.ID]wire.Value{testID("150"): {Tag: 5, Bytes: []byte("observability.log")}}},
	}}
	for _, name := range []string{"go-consumed:bytes:Equal", "go-consumed:maps:Clone", "go-consumed:slices:Clone,Replace", "go-consumed:log:Print"} {
		if !realizationPresent(e, name) {
			t.Fatalf("valid realization absent: %s", name)
		}
	}
	for _, name := range []string{"go-consumed:maps:Delete", "go-consumed:slices:Clone", "go-consumed:log:Printf", "go-consumed:strings:Clone"} {
		if realizationPresent(e, name) {
			t.Fatalf("unknown realization accepted: %s", name)
		}
	}
	delete(e.Entities, testID("101"))
	if realizationPresent(e, "go-consumed:maps:Clone") {
		t.Fatal("maps import accepted without canonical map update")
	}
	q := e.Entities[testID("103")]
	q.Fields[testID("150")] = wire.Value{Tag: 5, Bytes: []byte("other")}
	e.Entities[testID("103")] = q
	if realizationPresent(e, "go-consumed:log:Print") {
		t.Fatal("log import accepted without exact Foundation effect")
	}
}

func inventoryFixture(t *testing.T) (goprovider.DocumentSnapshot, goprovider.ResolutionManifest, contractcatalog.Contract, []byte, []byte) {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	exec, pkg := read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme")
	v1, e := contractcatalog.ResolveProjectContractSet(exec, pkg, read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	v2, e := contractcatalog.ResolveProjectContractSetV2(exec, pkg, read("../../../modules/project/v2/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	typ, param, body, fn, program := testID("8001"), testID("8002"), testID("8003"), testID("8004"), testID("8005")
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	blob := func(x string) wire.Value { return wire.Value{Tag: 5, Bytes: []byte(x)} }
	list := func(x ...wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
	execution := wire.Envelope{Module: testID("9000"), Revision: testID("8123"), Entities: map[wire.ID]wire.Entity{
		typ:     {ID: typ, Schema: testID("9010"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9100"): {Tag: 3, Unsigned: 64}, testID("9101"): {Tag: 2}, testID("9102"): {Tag: 3}}},
		param:   {ID: param, Schema: testID("9012"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9120"): blob("v"), testID("9121"): ref(typ), testID("9122"): {Tag: 3}}},
		body:    {ID: body, Schema: testID("9013"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9130"): ref(param)}},
		fn:      {ID: fn, Schema: testID("9011"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9110"): blob("Apply"), testID("9111"): list(ref(param)), testID("9112"): ref(typ), testID("9113"): ref(body)}},
		program: {ID: program, Schema: testID("9015"), Version: 1, Fields: map[wire.ID]wire.Value{testID("9150"): list(ref(fn)), testID("9151"): ref(fn)}},
	}}
	project, e := projectemitter.Emit(v1, projectemitter.Input{Identity: "example.test/evidence", RootPackage: "example.test/evidence", Execution: execution, Packages: []projectemitter.Package{{Name: "example.test/evidence", Interfaces: []projectemitter.Interface{{Name: "Apply", Function: fn, Parameters: []wire.ID{typ}, Result: typ}}}}})
	if e != nil {
		t.Fatal(e)
	}
	data := "package main\nfunc Apply(v int64) int64 { return v }\n"
	root := t.TempDir()
	if e = os.WriteFile(filepath.Join(root, "main.go"), []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
	source, e := projectsource.Discover(root, "example.test/evidence", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 2, MaxFileBytes: 1024, MaxTotalBytes: 1024})
	if e != nil {
		t.Fatal(e)
	}
	inv, e := sourceinventory.Emit(v2.Project(), project, source)
	if e != nil {
		t.Fatal(e)
	}
	snapshot := goprovider.DocumentSnapshot{Revision: 1, ModulePath: "example.test/evidence", PackagePath: "example.test/evidence", Entry: "Apply", Files: map[string]string{"main.go": data}}
	session, e := goprovider.NewIncrementalSession(read("../../../modules/execution/v35/module.g1"))
	if e != nil {
		t.Fatal(e)
	}
	result := session.Apply(snapshot)
	if !result.Valid {
		t.Fatalf("provider: %#v", result.Diagnostics)
	}
	return snapshot, result.Resolution, v2.Project(), project, inv
}

func TestEvidenceFromValidatedInventory(t *testing.T) {
	s, r, c, p, i := inventoryFixture(t)
	e, err := EvidenceFrom(s, r, c, p, i)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Sources) != 1 || len(e.Origins) != 1 {
		t.Fatalf("bad evidence: %#v", e)
	}
	if _, err = Convert(r, mustMetadata(t, s), e); err != nil {
		t.Fatal(err)
	}
}
func TestEvidenceRejectsDriftAndSpan(t *testing.T) {
	s, r, c, p, i := inventoryFixture(t)
	s.Files["main.go"] += "// drift"
	if _, e := EvidenceFrom(s, r, c, p, i); e == nil {
		t.Fatal("drift accepted")
	}
	s, r, c, p, i = inventoryFixture(t)
	r.Packages[0].Declarations[0].Location.ByteEnd = 9999
	if _, e := EvidenceFrom(s, r, c, p, i); e == nil {
		t.Fatal("span accepted")
	}
	s, r, c, p, i = inventoryFixture(t)
	s.Files["extra.go"] = "package x"
	if _, e := EvidenceFrom(s, r, c, p, i); e == nil {
		t.Fatal("extra accepted")
	}
	s, r, c, p, i = inventoryFixture(t)
	i[len(i)-1] ^= 1
	if _, e := EvidenceFrom(s, r, c, p, i); e == nil {
		t.Fatal("tampered inventory accepted")
	}
}
func mustMetadata(t *testing.T, s goprovider.DocumentSnapshot) []goprovider.PackageMetadata {
	t.Helper()
	read, err := os.ReadFile("../../../modules/execution/v35/module.g1")
	if err != nil {
		t.Fatal(err)
	}
	session, err := goprovider.NewIncrementalSession(read)
	if err != nil {
		t.Fatal(err)
	}
	r := session.Apply(s)
	if !r.Valid {
		t.Fatal(r.Diagnostics)
	}
	return r.Packages
}
func testID(x string) wire.ID {
	for len(x) < 32 {
		x = "0" + x
	}
	v, e := wire.ParseID(x)
	if e != nil {
		panic(e)
	}
	return v
}
