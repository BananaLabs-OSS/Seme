package projectemitter

import (
	"bytes"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectsnapshot"
	"seme.local/reference/wire"
)

func contracts(t *testing.T) contractcatalog.ProjectContractSet {
	t.Helper()
	read := func(path string) []byte {
		b, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	set, e := contractcatalog.ResolveProjectContractSet(read("../../../modules/execution/v35/module.seme"), read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if e != nil {
		t.Fatal(e)
	}
	return set
}

func execution() (wire.Envelope, wire.ID, wire.ID) {
	program := stableID("test", "program")
	function := stableID("test", "function")
	integer := stableID("test", "integer")
	parameter := stableID("test", "parameter")
	body := stableID("test", "body")
	return wire.Envelope{Module: id("00000000000000000000000000009000"), Entities: map[wire.ID]wire.Entity{
		program: {ID: program, Schema: programSchema, Version: 1, Fields: map[wire.ID]wire.Value{fProgramFunctions: list([]wire.Value{ref(function)}), fProgramEntry: ref(function)}},
		function: {ID: function, Schema: functionSchema, Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009110"): blob([]byte("Apply")), id("00000000000000000000000000009111"): list([]wire.Value{ref(parameter)}),
			id("00000000000000000000000000009112"): ref(integer), id("00000000000000000000000000009113"): ref(body),
		}},
		parameter: {ID: parameter, Schema: id("00000000000000000000000000009012"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009120"): blob([]byte("value")), id("00000000000000000000000000009121"): ref(integer), id("00000000000000000000000000009122"): {Tag: 4, Unsigned: 0},
		}},
		integer: {ID: integer, Schema: id("00000000000000000000000000009010"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009100"): {Tag: 4, Unsigned: 64}, id("00000000000000000000000000009101"): {Tag: 2}, id("00000000000000000000000000009102"): {Tag: 4, Unsigned: 0},
		}},
		body: {ID: body, Schema: id("00000000000000000000000000009070"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009700"): {Tag: 4, Unsigned: 0}, id("00000000000000000000000000009701"): ref(integer),
		}},
	}}, function, integer
}

func input() Input {
	exec, fn, integer := execution()
	return Input{Identity: "example.test/tool", RootPackage: "example.test/tool/app", Execution: exec, Packages: []Package{
		{Name: "example.test/tool/model", Interfaces: []Interface{{Name: "Apply", Function: fn, Parameters: []wire.ID{integer}, Result: integer}}},
		{Name: "example.test/tool/app", Dependencies: []Dependency{{Name: "model", Package: "example.test/tool/model"}}},
	}}
}

func TestEmitIsCanonicalAndOrderIndependent(t *testing.T) {
	set := contracts(t)
	a := input()
	first, e := Emit(set, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = projectsnapshot.Validate(first); e != nil {
		t.Fatal(e)
	}
	b := input()
	b.Packages[0], b.Packages[1] = b.Packages[1], b.Packages[0]
	second, e := Emit(set, b)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("ordering changed canonical project")
	}
	decoded, e := wire.Decode(first)
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := wire.Encode(decoded)
	if e != nil || !bytes.Equal(first, encoded) {
		t.Fatal("not canonical")
	}
}

func TestEmitContentMutationChangesProjectRevision(t *testing.T) {
	set := contracts(t)
	a := input()
	first, e := Emit(set, a)
	if e != nil {
		t.Fatal(e)
	}
	b := input()
	b.Packages[1].Dependencies[0].Name = "domain-model"
	second, e := Emit(set, b)
	if e != nil {
		t.Fatal(e)
	}
	if bytes.Equal(first, second) {
		t.Fatal("content mutation preserved project")
	}
	x, _ := wire.Decode(first)
	y, _ := wire.Decode(second)
	if x.Revision == y.Revision {
		t.Fatal("content mutation preserved revision")
	}
}

func TestEmitRejectsUnvalidatedContractsAndUnknownDependency(t *testing.T) {
	if _, e := Emit(contractcatalog.ProjectContractSet{}, input()); e == nil {
		t.Fatal("accepted unvalidated contracts")
	}
	in := input()
	in.Packages[1].Dependencies[0].Package = "example.test/missing"
	if _, e := Emit(contracts(t), in); e == nil {
		t.Fatal("accepted unknown dependency")
	}
}

func TestEmitRejectsInvalidProgramAndMultipleFunctionOwners(t *testing.T) {
	set := contracts(t)
	in := input()
	program, _ := soleProgram(in.Execution)
	p := in.Execution.Entities[program]
	delete(p.Fields, fProgramEntry)
	in.Execution.Entities[program] = p
	if _, e := Emit(set, in); e == nil {
		t.Fatal("accepted program without entry")
	}
	in = input()
	in.Packages = append(in.Packages, Package{Name: "example.test/tool/other", Interfaces: append([]Interface(nil), in.Packages[0].Interfaces...)})
	if _, e := Emit(set, in); e == nil {
		t.Fatal("accepted function owned by two packages")
	}
	in = input()
	in.Packages[0].Interfaces[0].Result = stableID("test", "wrong-result")
	if _, e := Emit(set, in); e == nil {
		t.Fatal("accepted interface that disagrees with function signature")
	}
	in = input()
	duplicate := in.Packages[0].Interfaces[0]
	duplicate.Function = stableID("test", "other-function")
	in.Packages[0].Interfaces = append(in.Packages[0].Interfaces, duplicate)
	if _, e := Emit(set, in); e == nil {
		t.Fatal("accepted duplicate exported interface name")
	}
}
