// Package projectv7instance binds a Project v6 snapshot to an authenticated
// Configuration v2 bound initialization graph.
package projectv7instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv6instance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts          contractcatalog.ProjectContractSetV7
	ProjectV6          projectv6instance.Inputs
	BoundConfiguration configurationinstance.BoundInput
	Composed           []byte
}

func Emit(in Inputs) ([]byte, error) {
	if err := components(in); err != nil {
		return nil, err
	}
	p, _ := wire.Decode(in.ProjectV6.Composed)
	c, _ := wire.Decode(in.BoundConfiguration.Artifact)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, src := range []wire.Envelope{p, c} {
		for x, q := range src.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !same(old, q) {
				return nil, fmt.Errorf("project_v7.component_collision:%s", x)
			}
			e.Entities[x] = clone(q)
		}
	}
	ps, err := one(p, id("e021"))
	if err != nil {
		return nil, err
	}
	cg, err := one(c, id("401f"))
	if err != nil {
		return nil, err
	}
	snapshot := stable(in.ProjectV6.Composed, in.BoundConfiguration.Artifact, "snapshot")
	module := stable(in.ProjectV6.Composed, in.BoundConfiguration.Artifact, "module")
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e022"), Version: 1, Fields: map[wire.ID]wire.Value{id("e220"): ref(ps), id("e221"): ref(cg), id("e222"): blob(make([]byte, 32))}}
	q := e.Entities[snapshot]
	q.Fields[id("e222")] = blob(SnapshotRevision(e, snapshot))
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b003")}, {Module: id("f000"), Revision: id("f001")}, {Module: id("4000"), Revision: id("4005")}, {Module: id("e000"), Revision: id("e009")}} {
		x := stable(in.ProjectV6.Composed, in.BoundConfiguration.Artifact, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("bound-configured-project-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snapshot)}}}}
	e.Revision = ArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Composed = out
	if err = Validate(in); err != nil {
		return nil, fmt.Errorf("project_v7.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(in Inputs) error {
	if err := components(in); err != nil {
		return err
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_v7.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, in.Composed) || e.Revision != ArtifactRevision(e) {
		return fmt.Errorf("project_v7.envelope")
	}
	m := e.Entities[e.Module]
	if m.Schema != id("12") || m.Version != 1 || string(m.Fields[id("120")].Bytes) != "bound-configured-project-v1" || !pins(e, m) {
		return fmt.Errorf("project_v7.module")
	}
	p, _ := wire.Decode(in.ProjectV6.Composed)
	c, _ := wire.Decode(in.BoundConfiguration.Artifact)
	snapshot, err := one(e, id("e022"))
	if err != nil {
		return err
	}
	if m.Fields[id("122")].Tag != 7 || len(m.Fields[id("122")].List) != 1 || m.Fields[id("122")].List[0].Reference != snapshot {
		return fmt.Errorf("project_v7.export")
	}
	ps, _ := one(p, id("e021"))
	cg, _ := one(c, id("401f"))
	q := e.Entities[snapshot]
	if q.Version != 1 || !shape(q, map[wire.ID]byte{id("e220"): 6, id("e221"): 6, id("e222"): 5}) || q.Fields[id("e220")].Reference != ps || q.Fields[id("e221")].Reference != cg || !bytes.Equal(q.Fields[id("e222")].Bytes, SnapshotRevision(e, snapshot)) {
		return fmt.Errorf("project_v7.snapshot")
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
			if got, ok := e.Entities[x]; !ok || !same(z, got) {
				return fmt.Errorf("project_v7.component:%s", x)
			}
			used[x] = true
		}
	}
	if len(used) != len(e.Entities) {
		return fmt.Errorf("project_v7.orphan")
	}
	return nil
}
func components(in Inputs) error {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e009")}) || in.Contracts.Configuration().Pin() != (contractcatalog.Pin{Module: id("4000"), Revision: id("4005")}) {
		return fmt.Errorf("project_v7.contracts")
	}
	if err := projectv6instance.ValidateBindable(in.ProjectV6); err != nil {
		return fmt.Errorf("project_v7.project_v6:%w", err)
	}
	if !bytes.Equal(in.BoundConfiguration.Base.ProjectV5.Composed, in.ProjectV6.ProjectV5.Composed) || !bytes.Equal(in.BoundConfiguration.Base.Artifact, in.ProjectV6.Configuration.Artifact) {
		return fmt.Errorf("project_v7.component_mismatch")
	}
	in.BoundConfiguration.Contracts = in.Contracts
	if err := configurationinstance.ValidateBound(in.BoundConfiguration); err != nil {
		return fmt.Errorf("project_v7.configuration:%w", err)
	}
	return nil
}
func pins(e wire.Envelope, m wire.Entity) bool {
	v := m.Fields[id("121")]
	want := map[wire.ID]wire.ID{id("9000"): id("9023"), id("b000"): id("b003"), id("f000"): id("f001"), id("4000"): id("4005"), id("e000"): id("e009")}
	if v.Tag != 7 || len(v.List) != len(want) {
		return false
	}
	got := map[wire.ID]wire.ID{}
	var prior wire.ID
	for i, r := range v.List {
		if r.Tag != 6 || i > 0 && bytes.Compare(prior[:], r.Reference[:]) >= 0 {
			return false
		}
		prior = r.Reference
		q := e.Entities[r.Reference]
		if q.Schema != id("13") || !shape(q, map[wire.ID]byte{id("130"): 6, id("131"): 5}) || len(q.Fields[id("131")].Bytes) != 16 {
			return false
		}
		var revision wire.ID
		copy(revision[:], q.Fields[id("131")].Bytes)
		if _, exists := got[q.Fields[id("130")].Reference]; exists {
			return false
		}
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
	q := clone(e.Entities[root])
	q.Fields[id("e222")] = blob(make([]byte, 32))
	entities := map[wire.ID]wire.Entity{}
	for x, v := range e.Entities {
		entities[x] = v
	}
	entities[root] = q
	b, _ := wire.Encode(wire.Envelope{Entities: closure(entities, root)})
	h := sha256.Sum256(append([]byte("seme.project-v7.snapshot.v1\x00"), b...))
	return h[:]
}
func ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v7.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(a, b []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v7.identity.v1\x00"))
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
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return out, fmt.Errorf("project_v7.schema_count")
			}
			out = x
		}
	}
	if out == (wire.ID{}) {
		return out, fmt.Errorf("project_v7.schema_count")
	}
	return out, nil
}
func closure(es map[wire.ID]wire.Entity, root wire.ID) map[wire.ID]wire.Entity {
	out := map[wire.ID]wire.Entity{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if _, ok := out[x]; ok {
			continue
		}
		q, ok := es[x]
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
	if v.Tag == 7 {
		for _, x := range v.List {
			collect(x, out)
		}
	}
}
func clone(q wire.Entity) wire.Entity {
	fields := map[wire.ID]wire.Value{}
	for x, v := range q.Fields {
		fields[x] = copyValue(v)
	}
	q.Fields = fields
	return q
}
func copyValue(v wire.Value) wire.Value {
	v.Bytes = append([]byte(nil), v.Bytes...)
	v.List = append([]wire.Value(nil), v.List...)
	for i := range v.List {
		v.List[i] = copyValue(v.List[i])
	}
	return v
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Module: a.ID, Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Module: b.ID, Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func shape(q wire.Entity, want map[wire.ID]byte) bool {
	if len(q.Fields) != len(want) {
		return false
	}
	for x, t := range want {
		v, ok := q.Fields[x]
		if !ok || v.Tag != t {
			return false
		}
	}
	return true
}
func sortRefs(xs []wire.Value) {
	sort.Slice(xs, func(i, j int) bool { return bytes.Compare(xs[i].Reference[:], xs[j].Reference[:]) < 0 })
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
