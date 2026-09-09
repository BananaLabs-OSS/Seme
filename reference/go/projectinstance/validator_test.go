package projectinstance_test

import (
	"bytes"
	"crypto/sha256"
	"os"
	"sort"
	"strings"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectemitter"
	"seme.local/reference/projectinstance"
	"seme.local/reference/wire"
)

var (
	moduleSchema  = tid("00000000000000000000000000000012")
	importSchema  = tid("00000000000000000000000000000013")
	fImports      = tid("00000000000000000000000000000121")
	fExports      = tid("00000000000000000000000000000122")
	fImportModule = tid("00000000000000000000000000000130")
	fImportRev    = tid("00000000000000000000000000000131")
	snapshot      = tid("0000000000000000000000000000e011")
)

func TestValidateAcceptsEmitterArtifact(t *testing.T) {
	source := validArtifact(t)
	if err := projectinstance.Validate(source); err != nil {
		t.Fatal(err)
	}
	decoded, err := wire.Decode(source)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := wire.Encode(decoded)
	if err != nil || !bytes.Equal(source, reencoded) {
		t.Fatal("accepted artifact was not canonical")
	}
}

func TestValidateRejectsImportAdversaries(t *testing.T) {
	tests := map[string]struct {
		want   string
		mutate func(*wire.Envelope)
	}{
		"missing": {"import_count", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fImports]
			delete(e.Entities, v.List[0].Reference)
			v.List = v.List[1:]
			m.Fields[fImports] = v
			e.Entities[e.Module] = m
		}},
		"extra": {"import_count", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fImports]
			x := tid("f1000000000000000000000000000001")
			e.Entities[x] = wire.Entity{ID: x, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{fImportModule: {Tag: 6, Reference: tid("f2000000000000000000000000000002")}, fImportRev: {Tag: 5, Bytes: make([]byte, 16)}}}
			v.List = append(v.List, wire.Value{Tag: 6, Reference: x})
			sort.Slice(v.List, func(i, j int) bool { return bytes.Compare(v.List[i].Reference[:], v.List[j].Reference[:]) < 0 })
			m.Fields[fImports] = v
			e.Entities[e.Module] = m
		}},
		"duplicate": {"import_reference", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fImports]
			v.List[1] = v.List[0]
			m.Fields[fImports] = v
			e.Entities[e.Module] = m
		}},
		"wrong-pin": {"import_pin", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			i := e.Entities[m.Fields[fImports].List[0].Reference]
			v := i.Fields[fImportRev]
			v.Bytes = append([]byte(nil), v.Bytes...)
			v.Bytes[15] ^= 1
			i.Fields[fImportRev] = v
			e.Entities[i.ID] = i
		}},
		"orphan": {"orphan_import", func(e *wire.Envelope) {
			x := tid("f3000000000000000000000000000003")
			e.Entities[x] = wire.Entity{ID: x, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{fImportModule: {Tag: 6, Reference: tid("f4000000000000000000000000000004")}, fImportRev: {Tag: 5, Bytes: make([]byte, 16)}}}
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			e, _ := wire.Decode(validArtifact(t))
			test.mutate(&e)
			source := sign(t, e)
			if err := projectinstance.Validate(source); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateRejectsModuleAndRevisionAdversaries(t *testing.T) {
	base := validArtifact(t)
	e, _ := wire.Decode(base)
	e.Revision[0] ^= 1
	encoded, _ := wire.Encode(e)
	if err := projectinstance.Validate(encoded); err == nil || !strings.Contains(err.Error(), "artifact_revision") {
		t.Fatalf("stale artifact revision error=%v", err)
	}

	e, _ = wire.Decode(base)
	for id, entity := range e.Entities {
		if entity.Schema != moduleSchema {
			e.Module = id
			break
		}
	}
	if err := projectinstance.Validate(sign(t, e)); err == nil || !strings.Contains(err.Error(), "module_declaration") {
		t.Fatalf("module mismatch error=%v", err)
	}

	e, _ = wire.Decode(base)
	x := tid("f5000000000000000000000000000005")
	e.Entities[x] = wire.Entity{ID: x, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{}}
	if err := projectinstance.Validate(sign(t, e)); err == nil || !strings.Contains(err.Error(), "module_count") {
		t.Fatalf("extra module error=%v", err)
	}

	e, _ = wire.Decode(base)
	for id, entity := range e.Entities {
		if entity.Schema == snapshot {
			v := entity.Fields[tid("0000000000000000000000000000e111")]
			v.Bytes = append([]byte(nil), v.Bytes...)
			v.Bytes[0] ^= 1
			entity.Fields[tid("0000000000000000000000000000e111")] = v
			e.Entities[id] = entity
		}
	}
	if err := projectinstance.Validate(sign(t, e)); err == nil || !strings.Contains(err.Error(), "project_instance.snapshot") {
		t.Fatalf("snapshot delegation error=%v", err)
	}
}

func TestValidateRejectsExportAdversaries(t *testing.T) {
	tests := map[string]struct {
		want   string
		mutate func(*wire.Envelope)
	}{
		"missing": {"export_set", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fExports]
			v.List = v.List[1:]
			m.Fields[fExports] = v
			e.Entities[e.Module] = m
		}},
		"duplicate": {"export_reference", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fExports]
			v.List[1] = v.List[0]
			m.Fields[fExports] = v
			e.Entities[e.Module] = m
		}},
		"import-target": {"export_target", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fExports]
			v.List[0] = m.Fields[fImports].List[0]
			sort.Slice(v.List, func(i, j int) bool { return bytes.Compare(v.List[i].Reference[:], v.List[j].Reference[:]) < 0 })
			m.Fields[fExports] = v
			e.Entities[e.Module] = m
		}},
		"unsorted": {"export_order", func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[fExports]
			v.List[0], v.List[len(v.List)-1] = v.List[len(v.List)-1], v.List[0]
			m.Fields[fExports] = v
			e.Entities[e.Module] = m
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			e, _ := wire.Decode(validArtifact(t))
			test.mutate(&e)
			err := projectinstance.Validate(sign(t, e))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateRejectsNoncanonicalOrMalformedWire(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("not-wire"), append(validArtifact(t), 0)} {
		if projectinstance.Validate(source) == nil {
			t.Fatal("accepted malformed/noncanonical bytes")
		}
	}
}

func validArtifact(t *testing.T) []byte {
	t.Helper()
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
	program := tid("71000000000000000000000000000001")
	function := tid("71000000000000000000000000000002")
	integer := tid("71000000000000000000000000000003")
	parameter := tid("71000000000000000000000000000004")
	body := tid("71000000000000000000000000000005")
	execution := wire.Envelope{Module: tid("00000000000000000000000000009000"), Entities: map[wire.ID]wire.Entity{
		program: {ID: program, Schema: tid("00000000000000000000000000009015"), Version: 1, Fields: map[wire.ID]wire.Value{tid("00000000000000000000000000009150"): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: function}}}, tid("00000000000000000000000000009151"): {Tag: 6, Reference: function}}},
		function: {ID: function, Schema: tid("00000000000000000000000000009011"), Version: 1, Fields: map[wire.ID]wire.Value{
			tid("00000000000000000000000000009110"): {Tag: 5, Bytes: []byte("Apply")}, tid("00000000000000000000000000009111"): {Tag: 7, List: []wire.Value{{Tag: 6, Reference: parameter}}},
			tid("00000000000000000000000000009112"): {Tag: 6, Reference: integer}, tid("00000000000000000000000000009113"): {Tag: 6, Reference: body},
		}},
		parameter: {ID: parameter, Schema: tid("00000000000000000000000000009012"), Version: 1, Fields: map[wire.ID]wire.Value{
			tid("00000000000000000000000000009120"): {Tag: 5, Bytes: []byte("value")}, tid("00000000000000000000000000009121"): {Tag: 6, Reference: integer}, tid("00000000000000000000000000009122"): {Tag: 3},
		}},
		integer: {ID: integer, Schema: tid("00000000000000000000000000009010"), Version: 1, Fields: map[wire.ID]wire.Value{
			tid("00000000000000000000000000009100"): {Tag: 3, Unsigned: 64}, tid("00000000000000000000000000009101"): {Tag: 2}, tid("00000000000000000000000000009102"): {Tag: 3},
		}},
		body: {ID: body, Schema: tid("00000000000000000000000000009070"), Version: 1, Fields: map[wire.ID]wire.Value{
			tid("00000000000000000000000000009700"): {Tag: 3}, tid("00000000000000000000000000009701"): {Tag: 6, Reference: integer},
		}},
	}}
	out, err := projectemitter.Emit(contracts, projectemitter.Input{Identity: "example.test/project-instance", RootPackage: "example.test/project-instance", Execution: execution, Packages: []projectemitter.Package{{Name: "example.test/project-instance", Interfaces: []projectemitter.Interface{{Name: "Apply", Function: function, Parameters: []wire.ID{integer}, Result: integer}}}}})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func sign(t *testing.T, e wire.Envelope) []byte {
	t.Helper()
	e.Revision = wire.ID{}
	unsigned, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	h.Write([]byte("seme.project.artifact.v1\x00"))
	h.Write(unsigned)
	copy(e.Revision[:], h.Sum(nil)[:16])
	out, err := wire.Encode(e)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func tid(value string) wire.ID {
	id, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}
