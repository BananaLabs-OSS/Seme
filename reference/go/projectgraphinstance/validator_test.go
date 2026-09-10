package projectgraphinstance

import (
	"bytes"
	"os"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/wire"
)

func TestRequiresAuthenticatedExactV3ContractSet(t *testing.T) {
	if err := validateContracts(contractcatalog.ProjectContractSet{}); err == nil {
		t.Fatal("accepted zero contract set")
	}
	read := func(path string) []byte {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	execution := read("../../../modules/execution/v35/module.seme")
	v1, err := contractcatalog.ResolveProjectContractSet(execution, read("../../../modules/package/v1/module.seme"), read("../../../modules/project/v1/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	if err = validateContracts(v1); err == nil {
		t.Fatal("accepted authenticated v1 set")
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(execution, read("../../../modules/package/v2/module.seme"), read("../../../modules/project/v3/module.seme"))
	if err != nil {
		t.Fatal(err)
	}
	if err = validateContracts(v3); err != nil {
		t.Fatalf("rejected authenticated v3 set: %v", err)
	}
}

func TestV3ModuleBoundaryRejectsPinExportAndOrphanForgeries(t *testing.T) {
	base := moduleFixture()
	if err := validateModule(base); err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*wire.Envelope){
		"wrong-package-pin": func(e *wire.Envelope) { mutateImport(e, id("b000"), id("b001")) },
		"wrong-project-pin": func(e *wire.Envelope) { mutateImport(e, id("e000"), id("e002")) },
		"duplicate-pin": func(e *wire.Envelope) {
			mutateImport(e, id("e000"), id("b002"))
			for x, q := range e.Entities {
				if q.Schema == importSchema && q.Fields[id("130")].Reference == id("e000") {
					q.Fields[id("130")] = ref(id("b000"))
					e.Entities[x] = q
				}
			}
		},
		"orphan-import": func(e *wire.Envelope) { e.Entities[id("f101")] = importEntity(id("f101"), id("e000"), id("e003")) },
		"duplicate-export": func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[id("122")]
			v.List[1] = v.List[0]
			m.Fields[id("122")] = v
			e.Entities[m.ID] = m
		},
		"wrong-export-schema": func(e *wire.Envelope) { x := e.Entities[id("f002")]; x.Schema = id("e017"); e.Entities[x.ID] = x },
		"unsorted-imports": func(e *wire.Envelope) {
			m := e.Entities[e.Module]
			v := m.Fields[id("121")]
			v.List[0], v.List[1] = v.List[1], v.List[0]
			m.Fields[id("121")] = v
			e.Entities[m.ID] = m
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			e := clone(base)
			mutate(&e)
			if validateModule(e) == nil {
				t.Fatal("accepted forged v3 boundary")
			}
		})
	}
}

func TestV3RevisionsAreDeterministicDomainSeparatedAndTransitive(t *testing.T) {
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	leaf := id("f001")
	binding := id("f002")
	snapshot := id("f003")
	e.Entities[leaf] = wire.Entity{ID: leaf, Schema: id("e016"), Version: 1, Fields: map[wire.ID]wire.Value{id("e160"): {Tag: 5, Bytes: bytes.Repeat([]byte{1}, 32)}}}
	e.Entities[binding] = wire.Entity{ID: binding, Schema: bindingSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e170"): ref(leaf), id("e171"): ref(leaf), id("e172"): {Tag: 5, Bytes: make([]byte, 32)}}}
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: graphSnapshotSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("e180"): ref(leaf), id("e181"): ref(binding), id("e182"): {Tag: 5, Bytes: make([]byte, 32)}}}
	a, err := BindingRevision(e, binding)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := BindingRevision(e, binding)
	if !bytes.Equal(a, b) {
		t.Fatal("binding revision nondeterministic")
	}
	c, err := SnapshotRevision(e, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, c) {
		t.Fatal("revision domains collide")
	}
	q := e.Entities[leaf]
	q.Fields[id("e160")] = wire.Value{Tag: 5, Bytes: bytes.Repeat([]byte{2}, 32)}
	e.Entities[leaf] = q
	d, _ := BindingRevision(e, binding)
	if bytes.Equal(a, d) {
		t.Fatal("binding revision ignored transitive source inventory")
	}
	delete(e.Entities, leaf)
	if _, err = SnapshotRevision(e, snapshot); err == nil {
		t.Fatal("revision accepted dangling transitive reference")
	}
}

func TestComposedUnionRejectsDriftCollisionAndOrphans(t *testing.T) {
	x, b, s := id("f020"), id("f021"), id("f022")
	entity := wire.Entity{ID: x, Schema: id("9010"), Version: 1, Fields: map[wire.ID]wire.Value{id("9100"): {Tag: 3, Unsigned: 64}}}
	composed := wire.Envelope{Module: id("f023"), Entities: map[wire.ID]wire.Entity{x: entity, b: {ID: b, Schema: bindingSchema}, s: {ID: s, Schema: graphSnapshotSchema}}}
	sources := []wire.Envelope{{Entities: map[wire.ID]wire.Entity{x: entity}}}
	if err := exactUnion(composed, sources, b, s); err != nil {
		t.Fatal(err)
	}
	drift := clone(composed)
	q := drift.Entities[x]
	q.Version = 2
	drift.Entities[x] = q
	if exactUnion(drift, sources, b, s) == nil {
		t.Fatal("accepted component drift")
	}
	collisionSource := clone(sources[0])
	q = collisionSource.Entities[x]
	q.Version = 2
	collisionSource.Entities[x] = q
	if exactUnion(composed, append(sources, collisionSource), b, s) == nil {
		t.Fatal("accepted component identity collision")
	}
	orphan := clone(composed)
	orphan.Entities[id("f024")] = wire.Entity{ID: id("f024"), Schema: id("9010"), Version: 1}
	if exactUnion(orphan, sources, b, s) == nil {
		t.Fatal("accepted orphan entity")
	}
}

func moduleFixture() wire.Envelope {
	m, b, s := id("f010"), id("f001"), id("f002")
	imports := []wire.ID{id("f011"), id("f012"), id("f013")}
	e := wire.Envelope{Module: m, Entities: map[wire.ID]wire.Entity{}}
	e.Entities[b] = wire.Entity{ID: b, Schema: bindingSchema, Version: 1}
	e.Entities[s] = wire.Entity{ID: s, Schema: graphSnapshotSchema, Version: 1}
	e.Entities[imports[0]] = importEntity(imports[0], id("9000"), id("9023"))
	e.Entities[imports[1]] = importEntity(imports[1], id("b000"), id("b002"))
	e.Entities[imports[2]] = importEntity(imports[2], id("e000"), id("e003"))
	e.Entities[m] = wire.Entity{ID: m, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): {Tag: 5, Bytes: []byte("project-graph-v1")}, id("121"): {Tag: 7, List: []wire.Value{ref(imports[0]), ref(imports[1]), ref(imports[2])}}, id("122"): {Tag: 7, List: []wire.Value{ref(b), ref(s)}}}}
	return e
}
func importEntity(x, module, revision wire.ID) wire.Entity {
	return wire.Entity{ID: x, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(module), id("131"): {Tag: 5, Bytes: append([]byte(nil), revision[:]...)}}}
}
func mutateImport(e *wire.Envelope, module, revision wire.ID) {
	for x, q := range e.Entities {
		if q.Schema == importSchema && q.Fields[id("130")].Reference == module {
			q.Fields[id("131")] = wire.Value{Tag: 5, Bytes: append([]byte(nil), revision[:]...)}
			e.Entities[x] = q
			return
		}
	}
}
func clone(e wire.Envelope) wire.Envelope {
	raw, _ := wire.Encode(e)
	out, _ := wire.Decode(raw)
	return out
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
