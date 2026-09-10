// Package projectv5instance composes authenticated Project v4 and Package v3
// instance graphs without redefining either graph's semantics.
package projectv5instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts            contractcatalog.ProjectContractSetV5
	ProjectV4            projectdependencyinstance.Inputs
	PackageV2, PackageV3 []byte
	Composed             []byte
}

func Emit(in Inputs) ([]byte, error) {
	if err := components(in); err != nil {
		return nil, err
	}
	v4, _ := wire.Decode(in.ProjectV4.Composed)
	p3, _ := wire.Decode(in.PackageV3)
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, src := range []wire.Envelope{p3, v4} {
		for x, q := range src.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if prior, ok := e.Entities[x]; ok {
				if !same(prior, q) {
					return nil, fmt.Errorf("project_v5.component_collision:%s", x)
				}
				continue
			}
			e.Entities[x] = clone(q)
		}
	}
	v4snap := one(v4, id("e019"))
	p3graph := one(p3, id("b029"))
	snapshot, module := Identity(in.ProjectV4.Composed, in.PackageV3, "snapshot"), Identity(in.ProjectV4.Composed, in.PackageV3, "module")
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e020"), Version: 1, Fields: map[wire.ID]wire.Value{id("e200"): ref(v4snap), id("e201"): ref(p3graph), id("e202"): blob(make([]byte, 32))}}
	q := e.Entities[snapshot]
	q.Fields[id("e202")] = blob(SnapshotRevision(e, snapshot))
	e.Entities[snapshot] = q
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("9000"), Revision: id("9023")}, {Module: id("b000"), Revision: id("b003")}, {Module: id("f000"), Revision: id("f001")}, {Module: id("e000"), Revision: id("e005")}} {
		x := Identity(in.ProjectV4.Composed, in.PackageV3, "import:"+pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("complete-project-graph-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snapshot)}}}}
	e.Revision = ArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Composed = out
	if err = Validate(in); err != nil {
		return nil, fmt.Errorf("project_v5.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(in Inputs) error {
	if err := components(in); err != nil {
		return err
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_v5.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, in.Composed) {
		return fmt.Errorf("project_v5.noncanonical")
	}
	if e.Module != Identity(in.ProjectV4.Composed, in.PackageV3, "module") || e.Revision != ArtifactRevision(e) || len(e.Parents) != 0 {
		return fmt.Errorf("project_v5.envelope")
	}
	m := e.Entities[e.Module]
	if m.Schema != id("12") || m.Version != 1 || len(m.Fields) != 3 || string(m.Fields[id("120")].Bytes) != "complete-project-graph-v1" || len(m.Fields[id("121")].List) != 4 || len(m.Fields[id("122")].List) != 1 {
		return fmt.Errorf("project_v5.module")
	}
	pins := map[wire.ID]wire.ID{}
	used := map[wire.ID]bool{e.Module: true}
	var prior wire.ID
	for i, v := range m.Fields[id("121")].List {
		if v.Tag != 6 || (i > 0 && bytes.Compare(prior[:], v.Reference[:]) >= 0) {
			return fmt.Errorf("project_v5.import_order")
		}
		prior = v.Reference
		q := e.Entities[v.Reference]
		if q.Schema != id("13") || q.Version != 1 || len(q.Fields) != 2 || q.Fields[id("130")].Tag != 6 || q.Fields[id("131")].Tag != 5 || len(q.Fields[id("131")].Bytes) != 16 {
			return fmt.Errorf("project_v5.import")
		}
		var r wire.ID
		copy(r[:], q.Fields[id("131")].Bytes)
		pins[q.Fields[id("130")].Reference] = r
		if v.Reference != Identity(in.ProjectV4.Composed, in.PackageV3, "import:"+q.Fields[id("130")].Reference.String()) {
			return fmt.Errorf("project_v5.import_identity")
		}
		used[v.Reference] = true
	}
	want := map[wire.ID]wire.ID{id("9000"): id("9023"), id("b000"): id("b003"), id("f000"): id("f001"), id("e000"): id("e005")}
	if len(pins) != len(want) {
		return fmt.Errorf("project_v5.pins")
	}
	for x, r := range want {
		if pins[x] != r {
			return fmt.Errorf("project_v5.pins")
		}
	}
	v4, _ := wire.Decode(in.ProjectV4.Composed)
	p3, _ := wire.Decode(in.PackageV3)
	if m.Fields[id("122")].List[0].Tag != 6 {
		return fmt.Errorf("project_v5.exports")
	}
	snap := m.Fields[id("122")].List[0].Reference
	q := e.Entities[snap]
	if snap != Identity(in.ProjectV4.Composed, in.PackageV3, "snapshot") || q.Schema != id("e020") || q.Version != 1 || len(q.Fields) != 3 || q.Fields[id("e200")].Tag != 6 || q.Fields[id("e201")].Tag != 6 || q.Fields[id("e202")].Tag != 5 || q.Fields[id("e200")].Reference != one(v4, id("e019")) || q.Fields[id("e201")].Reference != one(p3, id("b029")) || !bytes.Equal(q.Fields[id("e202")].Bytes, SnapshotRevision(e, snap)) {
		return fmt.Errorf("project_v5.snapshot")
	}
	used[snap] = true
	expected := map[wire.ID]wire.Entity{}
	for _, src := range []wire.Envelope{p3, v4} {
		for x, z := range src.Entities {
			if z.Schema != id("12") && z.Schema != id("13") {
				if prior, ok := expected[x]; ok && !same(prior, z) {
					return fmt.Errorf("project_v5.component_mismatch")
				}
				expected[x] = z
			}
		}
	}
	for x, z := range expected {
		got, ok := e.Entities[x]
		if !ok || !same(z, got) {
			return fmt.Errorf("project_v5.component:%s", x)
		}
		used[x] = true
	}
	if len(used) != len(e.Entities) {
		return fmt.Errorf("project_v5.orphan")
	}
	return nil
}

func components(in Inputs) error {
	if !in.Contracts.Validated() || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e005")}) {
		return fmt.Errorf("project_v5.contracts")
	}
	if err := projectdependencyinstance.Validate(in.ProjectV4); err != nil {
		return fmt.Errorf("project_v5.v4:%w", err)
	}
	if err := packagev3instance.Validate(in.Contracts, in.PackageV2, in.PackageV3); err != nil {
		return fmt.Errorf("project_v5.package_v3:%w", err)
	}
	return nil
}
func Identity(a, b []byte, role string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v5.identity.v1\x00"))
	for _, raw := range [][]byte{a, b} {
		x := sha256.Sum256(raw)
		h.Write(x[:])
	}
	_ = binary.Write(h, binary.BigEndian, uint64(len(role)))
	h.Write([]byte(role))
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}
func SnapshotRevision(e wire.Envelope, x wire.ID) []byte {
	seen := map[wire.ID]bool{}
	todo := []wire.ID{x}
	closure := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for len(todo) > 0 {
		idv := todo[0]
		todo = todo[1:]
		if seen[idv] {
			continue
		}
		seen[idv] = true
		q, ok := e.Entities[idv]
		if !ok {
			return nil
		}
		fields := map[wire.ID]wire.Value{}
		for f, v := range q.Fields {
			if idv == x && f == id("e202") {
				continue
			}
			fields[f] = v
			collect(v, &todo)
		}
		q.Fields = fields
		closure.Entities[idv] = q
	}
	b, _ := wire.Encode(closure)
	h := sha256.Sum256(append([]byte("seme.complete-project-graph.v1\x00"), b...))
	return h[:]
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
func ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.project-v5.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func clone(q wire.Entity) wire.Entity {
	b, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{q.ID: q}})
	e, _ := wire.Decode(b)
	return e.Entities[q.ID]
}
func one(e wire.Envelope, s wire.ID) wire.ID {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return wire.ID{}
			}
			out = x
		}
	}
	return out
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
