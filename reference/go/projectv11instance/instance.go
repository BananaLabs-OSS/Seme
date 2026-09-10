// Package projectv11instance binds an exact Project-v10 snapshot to one
// authenticated project-owned Ordered Transport v1 plan.
package projectv11instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/orderedtransportinstance"
	"seme.local/reference/projectv10instance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV11
	ProjectV10 projectv10instance.Inputs
	Transport  orderedtransportinstance.Inputs
	Composed   []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV10: in.ProjectV10, Transport: in.Transport})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Composed) {
		return fmt.Errorf("project_v11.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: v11id("e000"), Revision: v11id("e030")}) {
		return nil, fmt.Errorf("project_v11.contracts")
	}
	if err := projectv10instance.Validate(in.ProjectV10); err != nil {
		return nil, fmt.Errorf("project_v11.project_v10:%w", err)
	}
	if !bytes.Equal(in.ProjectV10.Composed, in.Transport.ProjectV10.Composed) {
		return nil, fmt.Errorf("project_v11.mixed_project")
	}
	in.Transport.Contracts = in.Contracts
	if err := orderedtransportinstance.Validate(in.Transport); err != nil {
		return nil, fmt.Errorf("project_v11.transport:%w", err)
	}
	p, err := wire.Decode(in.ProjectV10.Composed)
	if err != nil {
		return nil, err
	}
	t, err := wire.Decode(in.Transport.Artifact)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range []wire.Envelope{p, t} {
		for x, q := range part.Entities {
			if q.Schema == v11id("12") || q.Schema == v11id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !v11same(old, q) {
				return nil, fmt.Errorf("project_v11.collision:%s", x)
			}
			e.Entities[x] = q
		}
	}
	base, err := v11one(e, v11id("e025"))
	if err != nil {
		return nil, err
	}
	plan, err := v11one(e, v11id("10100"))
	if err != nil {
		return nil, err
	}
	snap := v11stable(in, "snapshot")
	e.Entities[snap] = v11entity(snap, "e026", map[string]wire.Value{"e260": v11ref(base), "e261": v11ref(plan), "e262": v11blob(make([]byte, 32))})
	q := e.Entities[snap]
	q.Fields[v11id("e262")] = v11blob(v11ContentRevision(e, snap))
	e.Entities[snap] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: v11id("1000"), Revision: v11id("1001")}, {Module: v11id("10000"), Revision: v11id("10001")}, {Module: v11id("3000"), Revision: v11id("3001")}, {Module: v11id("4000"), Revision: v11id("4006")}, {Module: v11id("6000"), Revision: v11id("6001")}, {Module: v11id("8000"), Revision: v11id("8001")}, {Module: v11id("9000"), Revision: v11id("9024")}, {Module: v11id("b000"), Revision: v11id("b004")}, {Module: v11id("e000"), Revision: v11id("e030")}, {Module: v11id("f000"), Revision: v11id("f001")}} {
		x := v11stable(in, "import", pin.Module.String())
		e.Entities[x] = v11entity(x, "13", map[string]wire.Value{"130": v11ref(pin.Module), "131": v11blob(pin.Revision[:])})
		imports = append(imports, v11ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = v11stable(in, "module")
	e.Entities[e.Module] = v11entity(e.Module, "12", map[string]wire.Value{"120": v11blob([]byte("transport-project-v1")), "121": {Tag: 7, List: imports}, "122": v11list(snap)})
	e.Revision = v11ArtifactRevision(e)
	return wire.Encode(e)
}
func v11stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v11.identity.v1\x00"))
	for _, b := range [][]byte{in.ProjectV10.Composed, in.Transport.Artifact} {
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
func v11ContentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = v11clone(q.Fields)
	q.Fields[v11id("e262")] = v11blob(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: v11closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.project-v11.snapshot.v1\x00"), b...))
	return h[:]
}
func v11ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v11.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func v11one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	for k, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_v11.count:%s", s)
			}
			x = k
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_v11.count:%s", s)
	}
	return x, nil
}
func v11same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func v11entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[v11id(k)] = v
	}
	return wire.Entity{ID: x, Schema: v11id(s), Version: 1, Fields: f}
}
func v11ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func v11blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func v11list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, v11ref(x))
	}
	return v
}
func v11clone(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func v11collect(v wire.Value, o *[]wire.ID) {
	if v.Tag == 6 {
		*o = append(*o, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			v11collect(x, o)
		}
	}
	if v.Tag == 8 {
		for _, x := range v.Record {
			v11collect(x, o)
		}
	}
}
func v11closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
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
			v11collect(v, &todo)
		}
	}
	return o
}
func v11id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
