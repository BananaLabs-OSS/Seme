// Package projectv6instance binds an authenticated Project v5 graph to an
// authenticated immutable Configuration v1 structural plan.
package projectv6instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts     contractcatalog.ProjectContractSetV6
	ProjectV5     projectv5instance.Inputs
	Configuration configurationinstance.Input
	Composed      []byte
}

func Emit(in Inputs) ([]byte, error) {
	return emit(in, false)
}

// EmitBindable emits the Project v6 structural component used by Project v7
// when its Configuration v1 graph has parameterized initializers. It is not a
// standalone Project v6 plan; Project v7 must supply and validate all bindings.
func EmitBindable(in Inputs) ([]byte, error) {
	return emit(in, true)
}

func emit(in Inputs, allowParameterized bool) ([]byte, error) {
	if err := components(in, allowParameterized); err != nil {
		return nil, err
	}
	p, _ := wire.Decode(in.ProjectV5.Composed)
	c, _ := wire.Decode(in.Configuration.Artifact)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, src := range []wire.Envelope{p, c} {
		for x, q := range src.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if prior, ok := e.Entities[x]; ok {
				if !same(prior, q) {
					return nil, fmt.Errorf("project_v6.component_collision:%s", x)
				}
				continue
			}
			e.Entities[x] = clone(q)
		}
	}
	ps, err := one(p, id("e020"))
	if err != nil {
		return nil, err
	}
	cg, err := one(c, id("4010"))
	if err != nil {
		return nil, err
	}
	snapshot, module := stable(in.ProjectV5.Composed, in.Configuration.Artifact, "snapshot"), stable(in.ProjectV5.Composed, in.Configuration.Artifact, "module")
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e021"), Version: 1, Fields: map[wire.ID]wire.Value{id("e210"): ref(ps), id("e211"): ref(cg), id("e212"): blob(make([]byte, 32))}}
	q := e.Entities[snapshot]
	q.Fields[id("e212")] = blob(SnapshotRevision(e, snapshot))
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b003")}, {Module: id("f000"), Revision: id("f001")}, {Module: id("4000"), Revision: id("4001")}, {Module: id("e000"), Revision: id("e008")}} {
		x := stable(in.ProjectV5.Composed, in.Configuration.Artifact, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("configured-project-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snapshot)}}}}
	e.Revision = ArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Composed = out
	if err = validate(in, allowParameterized); err != nil {
		return nil, fmt.Errorf("project_v6.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(in Inputs) error {
	return validate(in, false)
}

// ValidateBindable validates a Project v6 structural component whose
// parameterized initialization is completed by Project v7.
func ValidateBindable(in Inputs) error {
	return validate(in, true)
}

func validate(in Inputs, allowParameterized bool) error {
	if err := components(in, allowParameterized); err != nil {
		return err
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_v6.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, in.Composed) || e.Revision != ArtifactRevision(e) {
		return fmt.Errorf("project_v6.envelope")
	}
	m := e.Entities[e.Module]
	if m.Schema != id("12") || m.Version != 1 || len(m.Fields) != 3 || string(m.Fields[id("120")].Bytes) != "configured-project-v1" || !pins(e, m) {
		return fmt.Errorf("project_v6.module")
	}
	p, _ := wire.Decode(in.ProjectV5.Composed)
	c, _ := wire.Decode(in.Configuration.Artifact)
	snapshot, err := one(e, id("e021"))
	if err != nil {
		return err
	}
	if m.Fields[id("122")].Tag != 7 || len(m.Fields[id("122")].List) != 1 || m.Fields[id("122")].List[0].Reference != snapshot {
		return fmt.Errorf("project_v6.export")
	}
	q := e.Entities[snapshot]
	ps, _ := one(p, id("e020"))
	cg, _ := one(c, id("4010"))
	if q.Version != 1 || len(q.Fields) != 3 || q.Fields[id("e210")].Reference != ps || q.Fields[id("e211")].Reference != cg || !bytes.Equal(q.Fields[id("e212")].Bytes, SnapshotRevision(e, snapshot)) {
		return fmt.Errorf("project_v6.snapshot")
	}
	used := map[wire.ID]bool{e.Module: true, snapshot: true}
	for _, r := range m.Fields[id("121")].List {
		used[r.Reference] = true
	}
	for _, src := range []wire.Envelope{p, c} {
		for x, z := range src.Entities {
			if z.Schema == id("12") || z.Schema == id("13") {
				continue
			}
			got, ok := e.Entities[x]
			if !ok || !same(z, got) {
				return fmt.Errorf("project_v6.component:%s", x)
			}
			used[x] = true
		}
	}
	if len(used) != len(e.Entities) {
		return fmt.Errorf("project_v6.orphan")
	}
	return nil
}
func components(in Inputs, allowParameterized bool) error {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e008")}) {
		return fmt.Errorf("project_v6.contracts")
	}
	if err := projectv5instance.Validate(in.ProjectV5); err != nil {
		return fmt.Errorf("project_v6.project_v5:%w", err)
	}
	in.Configuration.Contracts = in.Contracts
	in.Configuration.ProjectV5 = in.ProjectV5
	var err error
	if allowParameterized {
		err = configurationinstance.ValidateBindableBase(in.Configuration)
	} else {
		err = configurationinstance.Validate(in.Configuration)
	}
	if err != nil {
		return fmt.Errorf("project_v6.configuration:%w", err)
	}
	return nil
}
func pins(e wire.Envelope, m wire.Entity) bool {
	v := m.Fields[id("121")]
	if v.Tag != 7 || len(v.List) != 5 {
		return false
	}
	want := map[wire.ID]wire.ID{id("9000"): id("9023"), id("b000"): id("b003"), id("f000"): id("f001"), id("4000"): id("4001"), id("e000"): id("e008")}
	got := map[wire.ID]wire.ID{}
	var prior wire.ID
	for i, r := range v.List {
		if r.Tag != 6 || i > 0 && bytes.Compare(prior[:], r.Reference[:]) >= 0 {
			return false
		}
		prior = r.Reference
		q := e.Entities[r.Reference]
		if q.Schema != id("13") || q.Version != 1 || len(q.Fields) != 2 || q.Fields[id("130")].Tag != 6 || q.Fields[id("131")].Tag != 5 || len(q.Fields[id("131")].Bytes) != 16 {
			return false
		}
		var revision wire.ID
		copy(revision[:], q.Fields[id("131")].Bytes)
		got[q.Fields[id("130")].Reference] = revision
	}
	for x, r := range want {
		if got[x] != r {
			return false
		}
	}
	return len(got) == len(want)
}
func SnapshotRevision(e wire.Envelope, root wire.ID) []byte {
	q := e.Entities[root]
	fields := make(map[wire.ID]wire.Value, len(q.Fields))
	for key, value := range q.Fields {
		fields[key] = value
	}
	q.Fields = fields
	q.Fields[id("e212")] = blob(make([]byte, 32))
	entities := make(map[wire.ID]wire.Entity, len(e.Entities))
	for entityID, entity := range e.Entities {
		entities[entityID] = entity
	}
	entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(entities, root)})
	h := sha256.Sum256(append([]byte("seme.project-v6.snapshot.v1\x00"), b...))
	return h[:]
}
func ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v6.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(a, b []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v6.identity.v1\x00"))
	x := sha256.Sum256(a)
	y := sha256.Sum256(b)
	h.Write(x[:])
	h.Write(y[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var out wire.ID
	copy(out[:], h.Sum(nil))
	return out
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	for id, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_v6.schema_count")
			}
			x = id
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_v6.schema_count")
	}
	return x, nil
}
func closure(all map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	out := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := out[x]; ok {
			continue
		}
		q, ok := all[x]
		if !ok {
			continue
		}
		out[x] = q
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return out
}
func collect(v wire.Value, out *[]wire.ID) {
	if v.Tag == 6 {
		*out = append(*out, v.Reference)
	}
	for _, x := range v.List {
		collect(x, out)
	}
	for _, x := range v.Record {
		collect(x, out)
	}
}
func clone(q wire.Entity) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for x, v := range q.Fields {
		f[x] = v
	}
	q.Fields = f
	return q
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
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
