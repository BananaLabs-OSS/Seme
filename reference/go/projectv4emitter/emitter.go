// Package projectv4emitter composes validated Project v3 and Dependency v1
// artifacts into the neutral Project Contract v4 dependency snapshot.
package projectv4emitter

import (
	"bytes"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/wire"
)

type Input struct {
	Contracts  contractcatalog.ProjectContractSetV4
	ProjectV3  projectgraphinstance.Inputs
	Dependency []byte
}

var moduleSchema = id("12")
var importSchema = id("13")

func Emit(in Input) ([]byte, error) {
	if !in.Contracts.Validated() {
		return nil, fmt.Errorf("project_v4_emitter.contracts")
	}
	if err := projectgraphinstance.Validate(in.ProjectV3); err != nil {
		return nil, fmt.Errorf("project_v4_emitter.project_v3:%w", err)
	}
	if _, err := dependencyinstance.Validate(in.Contracts.Dependency(), in.Dependency); err != nil {
		return nil, fmt.Errorf("project_v4_emitter.dependency:%w", err)
	}
	p, _ := wire.Decode(in.ProjectV3.Composed)
	d, _ := wire.Decode(in.Dependency)
	graph, err := one(p, id("e018"))
	if err != nil {
		return nil, err
	}
	closure, err := one(d, id("f010"))
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, src := range []wire.Envelope{p, d} {
		for x, q := range src.Entities {
			if q.Schema == moduleSchema || q.Schema == importSchema {
				continue
			}
			if _, exists := e.Entities[x]; exists {
				return nil, fmt.Errorf("project_v4_emitter.component_collision:%s", x)
			}
			e.Entities[x] = clone(q)
		}
	}
	stable := func(role string) wire.ID {
		return projectdependencyinstance.Identity(in.ProjectV3.Composed, in.Dependency, role)
	}
	snapshot, module := stable("snapshot"), stable("module")
	if _, ok := e.Entities[snapshot]; ok {
		return nil, fmt.Errorf("project_v4_emitter.identity_collision")
	}
	if _, ok := e.Entities[module]; ok {
		return nil, fmt.Errorf("project_v4_emitter.identity_collision")
	}
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e019"), Version: 1, Fields: map[wire.ID]wire.Value{id("e190"): ref(graph), id("e191"): ref(closure), id("e192"): blob(make([]byte, 32))}}
	r, err := projectdependencyinstance.SnapshotRevision(e, snapshot)
	if err != nil {
		return nil, err
	}
	q := e.Entities[snapshot]
	q.Fields[id("e192")] = blob(r)
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b002")}, {Module: id("f000"), Revision: id("f001")}, {Module: id("e000"), Revision: id("e004")}} {
		iid := stable("import:" + pin.Module.String())
		if _, ok := e.Entities[iid]; ok {
			return nil, fmt.Errorf("project_v4_emitter.identity_collision")
		}
		e.Entities[iid] = wire.Entity{ID: iid, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(iid))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("project-dependency-graph-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snapshot)}}}}
	e.Revision, err = projectdependencyinstance.ArtifactRevision(e)
	if err != nil {
		return nil, err
	}
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	if err = projectdependencyinstance.Validate(projectdependencyinstance.Inputs{Contracts: in.Contracts, ProjectV3: in.ProjectV3, Dependency: in.Dependency, Composed: out}); err != nil {
		return nil, fmt.Errorf("project_v4_emitter.validate:%w", err)
	}
	return out, nil
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	n := 0
	for id, q := range e.Entities {
		if q.Schema == s {
			x = id
			n++
		}
	}
	if n != 1 {
		return x, fmt.Errorf("project_v4_emitter.schema_count:%s:%d", s, n)
	}
	return x, nil
}
func clone(q wire.Entity) wire.Entity {
	b, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{q.ID: q}})
	e, _ := wire.Decode(b)
	return e.Entities[q.ID]
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func sortRefs(v []wire.Value) {
	sort.Slice(v, func(i, j int) bool { return bytes.Compare(v[i].Reference[:], v[j].Reference[:]) < 0 })
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
