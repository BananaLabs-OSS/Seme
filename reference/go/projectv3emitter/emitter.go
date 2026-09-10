// Package projectv3emitter composes independently validated Project v1,
// SourceInventory v2, and PackageGraph v2 artifacts into Project v3.
package projectv3emitter

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectinstance"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

type Input struct {
	Contracts                        contractcatalog.ProjectContractSet
	ProjectV2                        contractcatalog.Contract
	Project, Inventory, PackageGraph []byte
}

var moduleSchema = id("12")
var importSchema = id("13")
var projectSchema = id("e011")
var inventorySchema = id("e016")
var packageGraphSchema = id("b020")

func Emit(in Input) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Execution().Pin() != (contractcatalog.Pin{Module: id("9000"), Revision: id("9023")}) || in.Contracts.Package().Pin() != (contractcatalog.Pin{Module: id("b000"), Revision: id("b002")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e003")}) {
		return nil, fmt.Errorf("project_v3_emitter.contracts")
	}
	if err := projectinstance.Validate(in.Project); err != nil {
		return nil, fmt.Errorf("project_v3_emitter.project:%w", err)
	}
	if err := sourceinventory.Validate(in.ProjectV2, in.Project, in.Inventory); err != nil {
		return nil, fmt.Errorf("project_v3_emitter.inventory:%w", err)
	}
	if err := packagedetailinstance.Validate(in.PackageGraph); err != nil {
		return nil, fmt.Errorf("project_v3_emitter.package:%w", err)
	}
	p, _ := wire.Decode(in.Project)
	s, _ := wire.Decode(in.Inventory)
	g, _ := wire.Decode(in.PackageGraph)
	project, err := one(p, projectSchema)
	if err != nil {
		return nil, err
	}
	inventory, err := one(s, inventorySchema)
	if err != nil {
		return nil, err
	}
	graph, err := one(g, packageGraphSchema)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	if err = mergeComponents(e.Entities, p, s, g); err != nil {
		return nil, err
	}
	seed := sha256.New()
	seed.Write([]byte("seme.project-v3-emitter.v1\x00"))
	for _, raw := range [][]byte{in.Project, in.Inventory, in.PackageGraph} {
		sum := sha256.Sum256(raw)
		seed.Write(sum[:])
	}
	key := seed.Sum(nil)
	stable := func(role string) wire.ID {
		h := sha256.New()
		h.Write([]byte("seme.project-v3-emitter.identity.v1\x00"))
		h.Write(key)
		h.Write([]byte(role))
		var x wire.ID
		copy(x[:], h.Sum(nil))
		return x
	}
	binding, snapshot, module := stable("binding"), stable("snapshot"), stable("module")
	if _, ok := e.Entities[binding]; ok {
		return nil, fmt.Errorf("project_v3_emitter.identity_collision")
	}
	if _, ok := e.Entities[snapshot]; ok {
		return nil, fmt.Errorf("project_v3_emitter.identity_collision")
	}
	if _, ok := e.Entities[module]; ok {
		return nil, fmt.Errorf("project_v3_emitter.identity_collision")
	}
	e.Entities[binding] = wire.Entity{ID: binding, Schema: id("e017"), Version: 1, Fields: map[wire.ID]wire.Value{id("e170"): ref(graph), id("e171"): ref(inventory), id("e172"): blob(make([]byte, 32))}}
	r, err := projectgraphinstance.BindingRevision(e, binding)
	if err != nil {
		return nil, err
	}
	q := e.Entities[binding]
	q.Fields[id("e172")] = blob(r)
	e.Entities[binding] = q
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e018"), Version: 1, Fields: map[wire.ID]wire.Value{id("e180"): ref(project), id("e181"): ref(binding), id("e182"): blob(make([]byte, 32))}}
	r, err = projectgraphinstance.SnapshotRevision(e, snapshot)
	if err != nil {
		return nil, err
	}
	q = e.Entities[snapshot]
	q.Fields[id("e182")] = blob(r)
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b002")}, {Module: id("e000"), Revision: id("e003")}} {
		iid := stable("import:" + pin.Module.String())
		if _, ok := e.Entities[iid]; ok {
			return nil, fmt.Errorf("project_v3_emitter.identity_collision")
		}
		e.Entities[iid] = wire.Entity{ID: iid, Schema: importSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(iid))
	}
	sortRefs(imports)
	exports := []wire.Value{ref(binding), ref(snapshot)}
	sortRefs(exports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: moduleSchema, Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("project-graph-v1")), id("121"): list(imports), id("122"): list(exports)}}
	rev, err := projectgraphinstance.ArtifactRevision(e)
	if err != nil {
		return nil, err
	}
	e.Revision = rev
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	if err = projectgraphinstance.Validate(projectgraphinstance.Inputs{Contracts: in.Contracts, ProjectV2: in.ProjectV2, Project: in.Project, Inventory: in.Inventory, PackageGraph: in.PackageGraph, Composed: out}); err != nil {
		return nil, fmt.Errorf("project_v3_emitter.validate:%w", err)
	}
	return out, nil
}
func mergeComponents(dst map[wire.ID]wire.Entity, sources ...wire.Envelope) error {
	for _, src := range sources {
		for x, q := range src.Entities {
			if q.Schema == moduleSchema || q.Schema == importSchema {
				continue
			}
			if old, ok := dst[x]; ok && !reflect.DeepEqual(old, q) {
				return fmt.Errorf("project_v3_emitter.component_collision:%s", x)
			}
			dst[x] = cloneEntity(q)
		}
	}
	return nil
}
func one(e wire.Envelope, schema wire.ID) (wire.ID, error) {
	var out wire.ID
	n := 0
	for x, q := range e.Entities {
		if q.Schema == schema {
			out = x
			n++
		}
	}
	if n != 1 {
		return wire.ID{}, fmt.Errorf("project_v3_emitter.root:%s:%d", schema, n)
	}
	return out, nil
}
func cloneEntity(q wire.Entity) wire.Entity {
	raw, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{q.ID: q}})
	x, _ := wire.Decode(raw)
	return x.Entities[q.ID]
}
func ref(x wire.ID) wire.Value       { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value       { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func list(x []wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
}
func id(x string) wire.ID {
	for len(x) < 32 {
		x = "0" + x
	}
	v, e := wire.ParseID(x)
	if e != nil {
		panic(e)
	}
	return v
}
