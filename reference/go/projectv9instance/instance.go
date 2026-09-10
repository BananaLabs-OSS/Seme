// Package projectv9instance binds one authenticated Project-v8 snapshot to one
// detached Resource-v1 manifest.
package projectv9instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv8instance"
	"seme.local/reference/resourceinstance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts contractcatalog.ProjectContractSetV9
	ProjectV8 projectv8instance.Inputs
	Resource  resourceinstance.Inputs
	Composed  []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV8: in.ProjectV8, Resource: in.Resource})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Composed) {
		return fmt.Errorf("project_v9.artifact")
	}
	return nil
}
func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00b")}) || in.Contracts.Resource().Pin() != (contractcatalog.Pin{Module: id("6000"), Revision: id("6001")}) {
		return nil, fmt.Errorf("project_v9.contracts")
	}
	in.Resource.Contracts = in.Contracts
	in.Resource.ProjectV8 = in.ProjectV8
	if err := resourceinstance.Validate(in.Resource); err != nil {
		return nil, fmt.Errorf("project_v9.resource:%w", err)
	}
	p, err := wire.Decode(in.ProjectV8.Composed)
	if err != nil {
		return nil, err
	}
	r, err := wire.Decode(in.Resource.Artifact)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range []wire.Envelope{p, r} {
		for x, q := range part.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !same(old, q) {
				return nil, fmt.Errorf("project_v9.collision:%s", x)
			}
			e.Entities[x] = q
		}
	}
	base, err := one(e, id("e023"))
	if err != nil {
		return nil, err
	}
	manifest, err := one(e, id("6010"))
	if err != nil {
		return nil, err
	}
	snap := stable(in, "snapshot")
	e.Entities[snap] = wire.Entity{ID: snap, Schema: id("e024"), Version: 1, Fields: map[wire.ID]wire.Value{id("e240"): ref(base), id("e241"): ref(manifest), id("e242"): blob(make([]byte, 32))}}
	q := e.Entities[snap]
	q.Fields[id("e242")] = blob(revision(e, snap))
	e.Entities[snap] = q
	pins := []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("4000"), Revision: id("4006")}, {Module: id("6000"), Revision: id("6001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00b")}, {Module: id("f000"), Revision: id("f001")}}
	imports := []wire.Value{}
	for _, pin := range pins {
		x := stable(in, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	module := stable(in, "module")
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("resource-project-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snap)}}}}
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}
func revision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = clone(q.Fields)
	q.Fields[id("e242")] = blob(make([]byte, 32))
	es := map[wire.ID]wire.Entity{}
	for x, v := range e.Entities {
		es[x] = v
	}
	es[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(es, root)})
	h := sha256.Sum256(append([]byte("seme.project-v9.snapshot.v1\x00"), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v9.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v9.identity.v1\x00"))
	for _, b := range [][]byte{in.ProjectV8.Composed, in.Resource.Artifact} {
		x := sha256.Sum256(b)
		h.Write(x[:])
	}
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("project_v9.count:%s", s)
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("project_v9.count:%s", s)
	}
	return out, nil
}
func closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	o := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := o[x]; ok {
			continue
		}
		q, ok := es[x]
		if !ok {
			continue
		}
		o[x] = q
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return o
}
func collect(v wire.Value, o *[]wire.ID) {
	if v.Tag == 6 {
		*o = append(*o, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			collect(x, o)
		}
	}
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func clone(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
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
