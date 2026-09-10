package projectv3emitter

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailemitter"
	"seme.local/reference/projectemitter"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectsource"
	"seme.local/reference/projectv3report"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

func fixture(t *testing.T) Input {
	t.Helper()
	read := func(p string) []byte {
		b, e := os.ReadFile(p)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	exec, p1, p2 := read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/package/v2/module.seme")
	v1, e := contractcatalog.ResolveProjectContractSet(exec, p1, read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	v2, e := contractcatalog.ResolveProjectContractSetV2(exec, p1, read("../../../modules/project/v2/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	v3, e := contractcatalog.ResolveProjectContractSetV3(exec, p2, read("../../../modules/project/v3/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	typ, param, body, fn, program := id("8101"), id("8102"), id("8103"), id("8104"), id("8105")
	execution := wire.Envelope{Module: id("9000"), Revision: id("9023"), Entities: map[wire.ID]wire.Entity{typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}, id("9101"): {Tag: 2}, id("9102"): {Tag: 3}}}, param: {ID: param, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9120"): blob([]byte("v")), id("9121"): ref(typ), id("9122"): {Tag: 3}}}, body: {ID: body, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(param)}}, fn: {ID: fn, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): blob([]byte("Apply")), id("9111"): {Tag: 7, List: []wire.Value{ref(param)}}, id("9112"): ref(typ), id("9113"): ref(body)}}, program: {ID: program, Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{id("9150"): {Tag: 7, List: []wire.Value{ref(fn)}}, id("9151"): ref(fn)}}}}
	project, e := projectemitter.Emit(v1, projectemitter.Input{Identity: "example.test/v3", RootPackage: "example.test/v3", Execution: execution, Packages: []projectemitter.Package{{Name: "example.test/v3", Interfaces: []projectemitter.Interface{{Name: "Apply", Function: fn, Parameters: []wire.ID{typ}, Result: typ}}}}})
	if e != nil {
		t.Fatal(e)
	}
	data := []byte("package v3\nfunc Apply(v int64) int64 { return v }\n")
	dir := t.TempDir()
	if e = os.WriteFile(filepath.Join(dir, "main.go"), data, 0600); e != nil {
		t.Fatal(e)
	}
	src, e := projectsource.Discover(dir, "example.test/v3", projectsource.Toolchain{Language: "go", Toolchain: "go1.26", Profile: "bounded-v1", SemanticRevision: "provider-v1"}, projectsource.Policy{TrackedExtensions: []string{".go"}, GeneratedHeader: []byte("// generated"), MaxFiles: 2, MaxFileBytes: 1024, MaxTotalBytes: 1024})
	if e != nil {
		t.Fatal(e)
	}
	inventory, e := sourceinventory.Emit(v2.Project(), project, src)
	if e != nil {
		t.Fatal(e)
	}
	base, _ := wire.Decode(project)
	inv, _ := wire.Decode(inventory)
	var unit wire.ID
	for x, q := range inv.Entities {
		if q.Schema == id("e015") {
			unit = x
		}
		if q.Schema != moduleSchema && q.Schema != importSchema {
			if old, exists := base.Entities[x]; exists && !reflectEntity(old, q) {
				t.Fatal("fixture component collision")
			}
			base.Entities[x] = q
		}
	}
	digest := sha256.Sum256(data)
	start := uint64(len("package v3\nfunc "))
	origin := packagedetail.Origin{SourceIdentity: unit.String(), Path: "main.go", ContentDigest: digest, ByteStart: start, ByteEnd: start + 5, StartLine: 2, StartColumn: 6, EndLine: 2, EndColumn: 11}
	detail := packagedetail.Graph{Packages: []packagedetail.Detail{{Identity: "example.test/v3", Root: true, Sources: []packagedetail.Source{{Identity: unit.String(), Path: "main.go", ContentDigest: digest, ByteSize: uint64(len(data))}}, Members: []packagedetail.Member{{Identity: fn.String(), Name: "Apply", ExportName: "Apply", Visibility: packagedetail.Public, Origin: origin, Callable: true, Parameters: []string{typ.String()}, Results: []string{typ.String()}}}}}}
	packageGraph, e := packagedetailemitter.Emit(base, detail)
	if e != nil {
		t.Fatal(e)
	}
	return Input{Contracts: v3, ProjectV2: v2.Project(), Project: project, Inventory: inventory, PackageGraph: packageGraph}
}

func TestEmitRepeatValidatedAndImmutable(t *testing.T) {
	in := fixture(t)
	before := append([]byte(nil), in.Project...)
	beforeInventory := append([]byte(nil), in.Inventory...)
	beforePackages := append([]byte(nil), in.PackageGraph...)
	a, e := Emit(in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Emit(in)
	if e != nil || string(a) != string(b) {
		t.Fatal("nondeterministic")
	}
	if string(before) != string(in.Project) || string(beforeInventory) != string(in.Inventory) || string(beforePackages) != string(in.PackageGraph) {
		t.Fatal("input mutated")
	}
	if e = projectgraphinstance.Validate(projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: a}); e != nil {
		t.Fatal(e)
	}
	report, e := projectv3report.Inspect(projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: a})
	if e != nil || report.Root != "example.test/v3" || len(report.Packages) != 1 || len(report.Packages[0].Members) != 1 || report.Packages[0].Members[0].Visibility != "public" || report.Packages[0].Members[0].Export != "Apply" || len(report.Sources) != 1 || report.Sources[0].Digest == "" {
		t.Fatalf("report=%#v err=%v", report, e)
	}
	again, e := projectv3report.Inspect(projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: a})
	if e != nil || !reflect.DeepEqual(report, again) {
		t.Fatal("report nondeterministic")
	}
}
func TestMergeRejectsDivergentCollision(t *testing.T) {
	x := id("f900")
	a := wire.Entity{ID: x, Schema: id("9010"), Version: 1}
	b := a
	b.Version = 2
	if err := mergeComponents(map[wire.ID]wire.Entity{}, wire.Envelope{Entities: map[wire.ID]wire.Entity{x: a}}, wire.Envelope{Entities: map[wire.ID]wire.Entity{x: b}}); err == nil {
		t.Fatal("collision accepted")
	}
}
func TestEmitRejectsTamperCollisionWrongContract(t *testing.T) {
	in := fixture(t)
	composed, err := Emit(in)
	if err != nil {
		t.Fatal(err)
	}
	reportInput := projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: append([]byte(nil), composed...)}
	reportInput.Composed[len(reportInput.Composed)-1] ^= 1
	if _, err = projectv3report.Inspect(reportInput); err == nil {
		t.Fatal("report accepted tampered graph")
	}
	bad := in
	bad.Inventory = append([]byte(nil), bad.Inventory...)
	bad.Inventory[len(bad.Inventory)-1] ^= 1
	if _, e := Emit(bad); e == nil {
		t.Fatal("tamper accepted")
	}
	bad = in
	bad.Contracts = contractcatalog.ProjectContractSet{}
	if _, e := Emit(bad); e == nil {
		t.Fatal("contract accepted")
	}
	bad = in
	p, _ := wire.Decode(bad.Project)
	g, _ := wire.Decode(bad.PackageGraph)
	for x, q := range p.Entities {
		if old, ok := g.Entities[x]; ok && q.Schema != moduleSchema && q.Schema != importSchema {
			old.Version = q.Version + 1
			g.Entities[x] = old
			break
		}
	}
	bad.PackageGraph, _ = wire.Encode(g)
	if _, e := Emit(bad); e == nil {
		t.Fatal("collision accepted")
	}
}
func reflectEntity(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return string(x) == string(y)
}
