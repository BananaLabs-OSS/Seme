// Package projectv12instance binds an exact Project-v11 snapshot to one
// authenticated project-owned Controlled Effects v1 plan.
package projectv12instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV12
	ProjectV11 projectv11instance.Inputs
	Effects    controlledeffectsinstance.Inputs
	Composed   []byte
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV11: in.ProjectV11, Effects: in.Effects})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Composed) {
		return fmt.Errorf("project_v12.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: v12id("e000"), Revision: v12id("e031")}) {
		return nil, fmt.Errorf("project_v12.contracts")
	}
	if err := projectv11instance.Validate(in.ProjectV11); err != nil {
		return nil, fmt.Errorf("project_v12.project_v11:%w", err)
	}
	if !bytes.Equal(in.ProjectV11.Composed, in.Effects.ProjectV11.Composed) {
		return nil, fmt.Errorf("project_v12.mixed_project")
	}
	in.Effects.Contracts = in.Contracts
	if err := controlledeffectsinstance.Validate(in.Effects); err != nil {
		return nil, fmt.Errorf("project_v12.effects:%w", err)
	}
	p, err := wire.Decode(in.ProjectV11.Composed)
	if err != nil {
		return nil, err
	}
	c, err := wire.Decode(in.Effects.Artifact)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range []wire.Envelope{p, c} {
		for x, q := range part.Entities {
			if q.Schema == v12id("12") || q.Schema == v12id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !v12same(old, q) {
				return nil, fmt.Errorf("project_v12.collision:%s", x)
			}
			e.Entities[x] = q
		}
	}
	base, err := v12one(e, v12id("e026"))
	if err != nil {
		return nil, err
	}
	effects, err := v12one(e, v12id("13100"))
	if err != nil {
		return nil, err
	}
	snapshot := v12stable(in, "snapshot")
	e.Entities[snapshot] = v12entity(snapshot, "e029", map[string]wire.Value{"e290": v12ref(base), "e291": v12ref(effects), "e292": v12blob(make([]byte, 32))})
	q := e.Entities[snapshot]
	q.Fields[v12id("e292")] = v12blob(v12ContentRevision(e, snapshot))
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: v12id("1000"), Revision: v12id("1001")}, {Module: v12id("10000"), Revision: v12id("10001")}, {Module: v12id("13000"), Revision: v12id("13001")}, {Module: v12id("3000"), Revision: v12id("3001")}, {Module: v12id("4000"), Revision: v12id("4006")}, {Module: v12id("6000"), Revision: v12id("6001")}, {Module: v12id("8000"), Revision: v12id("8001")}, {Module: v12id("9000"), Revision: v12id("9024")}, {Module: v12id("b000"), Revision: v12id("b004")}, {Module: v12id("e000"), Revision: v12id("e031")}, {Module: v12id("f000"), Revision: v12id("f001")}} {
		x := v12stable(in, "import", pin.Module.String())
		e.Entities[x] = v12entity(x, "13", map[string]wire.Value{"130": v12ref(pin.Module), "131": v12blob(pin.Revision[:])})
		imports = append(imports, v12ref(x))
	}
	sort.Slice(imports, func(i, j int) bool { return bytes.Compare(imports[i].Reference[:], imports[j].Reference[:]) < 0 })
	e.Module = v12stable(in, "module")
	e.Entities[e.Module] = v12entity(e.Module, "12", map[string]wire.Value{"120": v12blob([]byte("effect-project-v1")), "121": {Tag: 7, List: imports}, "122": v12list(snapshot)})
	e.Revision = v12ArtifactRevision(e)
	return wire.Encode(e)
}
func v12stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v12.identity.v1\x00"))
	for _, b := range [][]byte{in.ProjectV11.Composed, in.Effects.Artifact} {
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
func v12ContentRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	q.Fields = v12clone(q.Fields)
	q.Fields[v12id("e292")] = v12blob(make([]byte, 32))
	e.Entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: v12closure(e.Entities, root)})
	h := sha256.Sum256(append([]byte("seme.project-v12.snapshot.v1\x00"), b...))
	return h[:]
}
func v12ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v12.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func v12one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	for k, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_v12.count:%s", s)
			}
			x = k
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_v12.count:%s", s)
	}
	return x, nil
}
func v12same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func v12entity(x wire.ID, s string, fs map[string]wire.Value) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for k, v := range fs {
		f[v12id(k)] = v
	}
	return wire.Entity{ID: x, Schema: v12id(s), Version: 1, Fields: f}
}
func v12ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func v12blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func v12list(xs ...wire.ID) wire.Value {
	v := wire.Value{Tag: 7}
	for _, x := range xs {
		v.List = append(v.List, v12ref(x))
	}
	return v
}
func v12clone(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
	o := map[wire.ID]wire.Value{}
	for k, v := range x {
		o[k] = v
	}
	return o
}
func v12collect(v wire.Value, o *[]wire.ID) {
	if v.Tag == 6 {
		*o = append(*o, v.Reference)
	}
	if v.Tag == 7 {
		for _, x := range v.List {
			v12collect(x, o)
		}
	}
	if v.Tag == 8 {
		for _, x := range v.Record {
			v12collect(x, o)
		}
	}
}
func v12closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
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
			v12collect(v, &todo)
		}
	}
	return o
}
func v12id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, e := wire.ParseID(s)
	if e != nil {
		panic(e)
	}
	return x
}
