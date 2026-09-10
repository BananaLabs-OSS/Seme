// Package projectv10instance binds an exact Project-v9 snapshot to a Durable-State-v1 plan.
package projectv10instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/durableinstance"
	"seme.local/reference/wire"
	"sort"
)

type Inputs struct {
	Contracts contractcatalog.ProjectContractSetV10
	ProjectV9 []byte
	Durable   durableinstance.Inputs
	Composed  []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV9: in.ProjectV9, Durable: in.Durable})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Composed) {
		return fmt.Errorf("project_v10.artifact")
	}
	return nil
}
func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00e")}) {
		return nil, fmt.Errorf("project_v10.contracts")
	}
	if !bytes.Equal(in.ProjectV9, in.Durable.ProjectV9) {
		return nil, fmt.Errorf("project_v10.mixed_project")
	}
	in.Durable.Contracts = in.Contracts
	if err := durableinstance.Validate(in.Durable); err != nil {
		return nil, fmt.Errorf("project_v10.durable:%w", err)
	}
	p, err := wire.Decode(in.ProjectV9)
	if err != nil {
		return nil, err
	}
	d, err := wire.Decode(in.Durable.Artifact)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range []wire.Envelope{p, d} {
		for x, q := range part.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !same(old, q) {
				return nil, fmt.Errorf("project_v10.collision:%s", x)
			}
			e.Entities[x] = q
		}
	}
	base, err := one(e, id("e024"))
	if err != nil {
		return nil, err
	}
	plan, err := one(e, id("8010"))
	if err != nil {
		return nil, err
	}
	snap := stable(in, "snapshot")
	e.Entities[snap] = entity(snap, "e025", map[string]wire.Value{"e250": ref(base), "e251": ref(plan), "e252": blob(make([]byte, 32))})
	q := e.Entities[snap]
	q.Fields[id("e252")] = blob(revision(e, snap))
	e.Entities[snap] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("4000"), Revision: id("4006")}, {Module: id("6000"), Revision: id("6001")}, {Module: id("8000"), Revision: id("8001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("e000"), Revision: id("e00e")}, {Module: id("f000"), Revision: id("f001")}} {
		x := stable(in, "import", pin.Module.String())
		e.Entities[x] = entity(x, "13", map[string]wire.Value{"130": ref(pin.Module), "131": blob(pin.Revision[:])})
		imports = append(imports, ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = stable(in, "module")
	e.Entities[e.Module] = entity(e.Module, "12", map[string]wire.Value{"120": blob([]byte("durable-project-v1")), "121": {Tag: 7, List: imports}, "122": list(snap)})
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}
func revision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = clone(q.Fields)
	q.Fields[id("e252")] = blob(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.project-v10.snapshot.v1\x00"), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v10.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v10.identity.v1\x00"))
	for _, b := range [][]byte{in.ProjectV9, in.Durable.Artifact} {
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
	var x wire.ID
	for k, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_v10.count:%s", s)
			}
			x = k
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_v10.count:%s", s)
	}
	return x, nil
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
func clone(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[id(k)] = v
	}
	return wire.Entity{ID: x, Schema: id(s), Version: 1, Fields: f}
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, ref(x))
	}
	return v
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
