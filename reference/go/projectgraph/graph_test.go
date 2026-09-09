package projectgraph

import (
	"bytes"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

func TestInspectReportsDeterministicValidatedGraph(t *testing.T) {
	artifact := testArtifact(t)
	a, err := InspectJSON(artifact)
	if err != nil {
		t.Fatal(err)
	}
	b, err := InspectJSON(artifact)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("nondeterministic report: %v", err)
	}
	report, err := Inspect(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if report.Profile != "seme.project.package-graph.v1" || len(report.Packages) != 2 {
		t.Fatalf("report=%#v", report)
	}
	leaf, root := report.Packages[0], report.Packages[1]
	if leaf.Name != "example.test/tool/model" || leaf.Root || len(leaf.Dependencies) != 0 || len(leaf.Exports) != 1 || leaf.Exports[0].Name != "Normalize" {
		t.Fatalf("leaf=%#v", leaf)
	}
	if root.Name != "example.test/tool/root" || !root.Root || len(root.Dependencies) != 1 || root.Dependencies[0] != (Dependency{Name: "model", Kind: "local", Package: leaf.Name}) || len(root.Exports) != 1 || root.Exports[0].Name != "Apply" {
		t.Fatalf("root=%#v", root)
	}
	if root.Exports[0].Parameters[0] != leaf.Exports[0].Parameters[0] || root.Exports[0].Result != leaf.Exports[0].Result {
		t.Fatal("signature identities were not preserved")
	}
}

func TestInspectRejectsMalformedAndTamperedArtifacts(t *testing.T) {
	if _, err := InspectJSON([]byte("not canonical wire")); err == nil {
		t.Fatal("accepted malformed artifact")
	}
	artifact := append([]byte(nil), testArtifact(t)...)
	artifact = append(artifact, 'x')
	if _, err := InspectJSON(artifact); err == nil {
		t.Fatal("accepted tampered artifact")
	}
}

func testArtifact(t *testing.T) []byte {
	t.Helper()
	read := func(path string) []byte {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	contracts, err := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	typ, p1, p2 := id("8001"), id("8002"), id("8003")
	b1, b2, f1, f2, program := id("8004"), id("8005"), id("8006"), id("8007"), id("8008")
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	list := func(xs ...wire.Value) wire.Value { return wire.Value{Tag: 7, List: xs} }
	parameter := func(pid wire.ID, name string) wire.Entity {
		return wire.Entity{ID: pid, Schema: id("9012"), Version: 1, Fields: map[wire.ID]wire.Value{id("9120"): {Tag: 5, Bytes: []byte(name)}, id("9121"): ref(typ), id("9122"): {Tag: 3}}}
	}
	function := func(fid, pid, body wire.ID, name string) wire.Entity {
		return wire.Entity{ID: fid, Schema: id("9011"), Version: 1, Fields: map[wire.ID]wire.Value{id("9110"): {Tag: 5, Bytes: []byte(name)}, id("9111"): list(ref(pid)), id("9112"): ref(typ), id("9113"): ref(body)}}
	}
	execution := wire.Envelope{Module: id("9000"), Revision: id("9023"), Entities: map[wire.ID]wire.Entity{
		typ: {ID: typ, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}, id("9101"): {Tag: 2}, id("9102"): {Tag: 3}}},
		p1:  parameter(p1, "value"), p2: parameter(p2, "value"),
		b1: {ID: b1, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(p1)}},
		b2: {ID: b2, Schema: id("9013"), Version: 1, Fields: map[wire.ID]wire.Value{id("9130"): ref(p2)}},
		f1: function(f1, p1, b1, "Normalize"), f2: function(f2, p2, b2, "Apply"),
		program: {ID: program, Schema: id("9015"), Version: 1, Fields: map[wire.ID]wire.Value{id("9150"): list(ref(f1), ref(f2)), id("9151"): ref(f2)}},
	}}
	artifact, err := projectemitter.Emit(contracts, projectemitter.Input{Identity: "example.test/tool", RootPackage: "example.test/tool/root", Execution: execution, Packages: []projectemitter.Package{
		{Name: "example.test/tool/model", Interfaces: []projectemitter.Interface{{Name: "Normalize", Function: f1, Parameters: []wire.ID{typ}, Result: typ}}},
		{Name: "example.test/tool/root", Dependencies: []projectemitter.Dependency{{Name: "model", Package: "example.test/tool/model"}}, Interfaces: []projectemitter.Interface{{Name: "Apply", Function: f2, Parameters: []wire.ID{typ}, Result: typ}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return artifact
}
