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

func contractsV8(t *testing.T) contractcatalog.ProjectContractSetV8 {
	t.Helper()
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	set, err := contractcatalog.ResolveProjectContractSetV8(read("../../../modules/foundation/v1/module.seme"), read("../../../modules/execution/v36/module.seme"), read("../../../modules/package/v4/module.seme"), read("../../../modules/dependency/v1/module.seme"), read("../../../modules/configuration/v3/module.seme"), read("../../../modules/project/v8/module.seme"))
	if err != nil {
		t.Fatal(err)
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
			id("00000000000000000000000000009120"): blob([]byte("value")), id("00000000000000000000000000009121"): ref(integer), id("00000000000000000000000000009122"): {Tag: 3, Unsigned: 0},
		}},
		integer: {ID: integer, Schema: id("00000000000000000000000000009010"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009100"): {Tag: 3, Unsigned: 64}, id("00000000000000000000000000009101"): {Tag: 2}, id("00000000000000000000000000009102"): {Tag: 3, Unsigned: 0},
		}},
		body: {ID: body, Schema: id("00000000000000000000000000009070"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("00000000000000000000000000009700"): {Tag: 3, Unsigned: 0}, id("00000000000000000000000000009701"): ref(integer),
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
	unused := stableID("test", "unused-provider-primitive")
	b.Execution.Entities[unused] = wire.Entity{ID: unused, Schema: id("00000000000000000000000000009020"), Version: 1, Fields: map[wire.ID]wire.Value{}}
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

func TestEmitV8BaseUsesOnlyExecutionV36Authority(t *testing.T) {
	in := input()
	first, err := EmitV8Base(contractsV8(t), in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EmitV8Base(contractsV8(t), in)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("v8 base is not deterministic")
	}
	e, err := wire.Decode(first)
	if err != nil {
		t.Fatal(err)
	}
	module := e.Entities[e.Module]
	foundV36 := false
	v36 := id("00000000000000000000000000009024")
	for _, value := range module.Fields[id("00000000000000000000000000000121")].List {
		imp := e.Entities[value.Reference]
		if imp.Fields[id("00000000000000000000000000000130")].Reference == id("00000000000000000000000000009000") {
			foundV36 = bytes.Equal(imp.Fields[id("00000000000000000000000000000131")].Bytes, v36[:])
		}
	}
	if !foundV36 {
		t.Fatal("v8 base omitted exact Execution v36 pin")
	}
	if _, err = EmitV8Base(contractcatalog.ProjectContractSetV8{}, in); err == nil {
		t.Fatal("accepted unvalidated v8 authority")
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

func TestEmitOwnsExactCanonicalEffectsAndRejectsForgeries(t *testing.T) {
	set := contracts(t)
	in := input()
	program, _ := soleProgram(in.Execution)
	function := in.Execution.Entities[program].Fields[fProgramEntry].Reference
	body := in.Execution.Entities[function].Fields[id("00000000000000000000000000009113")].Reference
	effect := stableID("test", "effect")
	capability := stableID("test", "capability")
	in.Execution.Entities[capability] = wire.Entity{ID: capability, Schema: capabilitySchema, Version: 1, Fields: map[wire.ID]wire.Value{id("00000000000000000000000000000160"): blob([]byte("observability.log"))}}
	in.Execution.Entities[effect] = wire.Entity{ID: effect, Schema: effectSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("00000000000000000000000000000150"): blob([]byte("observability.log")), id("00000000000000000000000000000151"): ref(capability)}}
	invoke := in.Execution.Entities[body]
	invoke.Schema = id("000000000000000000000000000090f1")
	invoke.Fields = map[wire.ID]wire.Value{id("00000000000000000000000000009f10"): ref(effect), id("00000000000000000000000000009f11"): list(nil)}
	in.Execution.Entities[body] = invoke
	in.Packages[0].Effects = []wire.ID{effect}
	first, err := Emit(set, in)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := wire.Decode(first)
	pid := stableID("package", in.Packages[0].Name)
	if got := decoded.Entities[pid].Fields[id("0000000000000000000000000000b104")].List; len(got) != 1 || got[0].Reference != effect {
		t.Fatalf("effect ownership missing: %#v", got)
	}

	for name, mutate := range map[string]func(*Input){
		"unowned":   func(x *Input) { x.Packages[0].Effects = nil },
		"duplicate": func(x *Input) { x.Packages[0].Effects = []wire.ID{effect, effect} },
		"foreign":   func(x *Input) { x.Packages[0].Effects = []wire.ID{stableID("test", "foreign")} },
	} {
		t.Run(name, func(t *testing.T) {
			bad := in
			bad.Packages = append([]Package(nil), in.Packages...)
			mutate(&bad)
			if out, err := Emit(set, bad); err == nil || out != nil {
				t.Fatalf("accepted forged effects: err=%v", err)
			}
		})
	}
	shared := in
	shared.Packages = append([]Package(nil), in.Packages...)
	shared.Packages[1].Effects = []wire.ID{effect}
	sharedOut, err := Emit(set, shared)
	if err != nil || len(sharedOut) == 0 {
		t.Fatalf("shared package requirement rejected: %v", err)
	}
	sharedAgain, err := Emit(set, shared)
	if err != nil || !bytes.Equal(sharedOut, sharedAgain) {
		t.Fatal("shared effect requirements were not deterministic")
	}
	malformed := in
	malformed.Execution.Entities = cloneExecutionEntities(in.Execution.Entities)
	broken := malformed.Execution.Entities[effect]
	delete(broken.Fields, id("00000000000000000000000000000151"))
	malformed.Execution.Entities[effect] = broken
	if out, err := Emit(set, malformed); err == nil || out != nil {
		t.Fatalf("malformed effect accepted: %v", err)
	}
}

func cloneExecutionEntities(in map[wire.ID]wire.Entity) map[wire.ID]wire.Entity {
	out := make(map[wire.ID]wire.Entity, len(in))
	for identity, entity := range in {
		fields := make(map[wire.ID]wire.Value, len(entity.Fields))
		for field, value := range entity.Fields {
			fields[field] = value
		}
		entity.Fields = fields
		out[identity] = entity
	}
	return out
}
