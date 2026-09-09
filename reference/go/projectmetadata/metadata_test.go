package projectmetadata

import (
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/wire"
)

func TestExtractUsesValidatedCanonicalOwnership(t *testing.T) {
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	contracts, err := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	typ, param, body, fn, program := mustID("8001"), mustID("8002"), mustID("8003"), mustID("8004"), mustID("8005")
	ref := func(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
	list := func(xs ...wire.Value) wire.Value { return wire.Value{Tag: 7, List: xs} }
	execution := wire.Envelope{Module: mustID("9000"), Revision: mustID("8123"), Entities: map[wire.ID]wire.Entity{
		typ:     {ID: typ, Schema: mustID("9010"), Version: 1, Fields: map[wire.ID]wire.Value{mustID("9100"): {Tag: 3, Unsigned: 64}, mustID("9101"): {Tag: 2}, mustID("9102"): {Tag: 3}}},
		param:   {ID: param, Schema: mustID("9012"), Version: 1, Fields: map[wire.ID]wire.Value{mustID("9120"): {Tag: 5, Bytes: []byte("value")}, mustID("9121"): ref(typ), mustID("9122"): {Tag: 3}}},
		body:    {ID: body, Schema: mustID("9013"), Version: 1, Fields: map[wire.ID]wire.Value{mustID("9130"): ref(param)}},
		fn:      {ID: fn, Schema: mustID("9011"), Version: 1, Fields: map[wire.ID]wire.Value{mustID("9110"): {Tag: 5, Bytes: []byte("Apply")}, mustID("9111"): list(ref(param)), mustID("9112"): ref(typ), mustID("9113"): ref(body)}},
		program: {ID: program, Schema: mustID("9015"), Version: 1, Fields: map[wire.ID]wire.Value{mustID("9150"): list(ref(fn)), mustID("9151"): ref(fn)}},
	}}
	artifact, err := projectemitter.Emit(contracts, projectemitter.Input{Identity: "example.test/project", RootPackage: "example.test/project/app", Execution: execution, Packages: []projectemitter.Package{{Name: "example.test/project/app", Interfaces: []projectemitter.Interface{{Name: "Apply", Function: fn, Parameters: []wire.ID{typ}, Result: typ}}}}})
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := Extract(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata) != 1 || !metadata[0].Root || metadata[0].Name != "example.test/project/app" || len(metadata[0].Functions) != 1 || metadata[0].Functions[0].ID != fn.String() || metadata[0].Functions[0].Name != "Apply" || metadata[0].Functions[0].Result != typ.String() {
		t.Fatalf("metadata=%#v", metadata)
	}
}

func TestExtractRejectsUnvalidatedBytes(t *testing.T) {
	if _, err := Extract([]byte("not a project")); err == nil {
		t.Fatal("accepted malformed project")
	}
}
