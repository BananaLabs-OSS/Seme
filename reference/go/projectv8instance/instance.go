// Package projectv8instance composes one complete, same-run v36 project
// snapshot without importing or upgrading any v35 project instance.
package projectv8instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	"seme.local/reference/configurationinstance"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/executionprofile"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

type Inputs struct {
	Contracts                contractcatalog.ProjectContractSetV8
	ProjectBase, Inventory   []byte
	PackageV2, PackageV4     []byte
	Dependency, Construction []byte
	Configuration            configurationinstance.V3Input
	Composed                 []byte
}

func Emit(in Inputs) ([]byte, error) {
	parts, roots, err := components(in)
	if err != nil {
		return nil, err
	}
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for _, part := range parts {
		for x, q := range part.Entities {
			if q.Schema == id("12") || q.Schema == id("13") {
				continue
			}
			if old, ok := e.Entities[x]; ok && !same(old, q) {
				return nil, fmt.Errorf("project_v8.component_collision:%s", x)
			}
			e.Entities[x] = clone(q)
		}
	}
	snapshot := stable(in, "snapshot")
	e.Entities[snapshot] = wire.Entity{ID: snapshot, Schema: id("e023"), Version: 1, Fields: map[wire.ID]wire.Value{id("e230"): ref(roots.inventory), id("e231"): ref(roots.dependency), id("e232"): ref(roots.packages), id("e233"): ref(roots.program), id("e234"): ref(roots.configuration), id("e235"): blob(make([]byte, 32))}}
	q := e.Entities[snapshot]
	q.Fields[id("e235")] = blob(SnapshotRevision(e, snapshot))
	e.Entities[snapshot] = q
	module := stable(in, "module")
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("3000"), Revision: id("3001")}, {Module: id("9000"), Revision: id("9024")}, {Module: id("b000"), Revision: id("b004")}, {Module: id("f000"), Revision: id("f001")}, {Module: id("4000"), Revision: id("4006")}, {Module: id("e000"), Revision: id("e00a")}} {
		x := stable(in, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("full-configured-project-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(snapshot)}}}}
	e.Revision = ArtifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	in.Composed = out
	if err = Validate(in); err != nil {
		return nil, fmt.Errorf("project_v8.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(in Inputs) error {
	parts, roots, err := components(in)
	if err != nil {
		return err
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_v8.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, in.Composed) || e.Revision != ArtifactRevision(e) {
		return fmt.Errorf("project_v8.envelope")
	}
	m := e.Entities[e.Module]
	if m.Schema != id("12") || m.Version != 1 || string(m.Fields[id("120")].Bytes) != "full-configured-project-v1" || !pins(e, m) {
		return fmt.Errorf("project_v8.module")
	}
	snapshot, err := one(e, id("e023"))
	if err != nil {
		return err
	}
	q := e.Entities[snapshot]
	if q.Version != 1 || !shape(q, map[wire.ID]byte{id("e230"): 6, id("e231"): 6, id("e232"): 6, id("e233"): 6, id("e234"): 6, id("e235"): 5}) || q.Fields[id("e230")].Reference != roots.inventory || q.Fields[id("e231")].Reference != roots.dependency || q.Fields[id("e232")].Reference != roots.packages || q.Fields[id("e233")].Reference != roots.program || q.Fields[id("e234")].Reference != roots.configuration || !bytes.Equal(q.Fields[id("e235")].Bytes, SnapshotRevision(e, snapshot)) {
		return fmt.Errorf("project_v8.snapshot")
	}
	if m.Fields[id("122")].Tag != 7 || len(m.Fields[id("122")].List) != 1 || m.Fields[id("122")].List[0].Reference != snapshot {
		return fmt.Errorf("project_v8.export")
	}
	if err = matchOrigins(e, roots.inventory, roots.packages); err != nil {
		return err
	}
	projectSnapshot, err := one(e, id("e011"))
	if err != nil {
		return fmt.Errorf("project_v8.project_snapshot:%w", err)
	}
	if err = matchPackages(e, projectSnapshot, roots.packages); err != nil {
		return err
	}
	used := map[wire.ID]bool{e.Module: true, snapshot: true}
	for _, r := range m.Fields[id("121")].List {
		used[r.Reference] = true
	}
	for _, part := range parts {
		for x, z := range part.Entities {
			if z.Schema == id("12") || z.Schema == id("13") {
				continue
			}
			got, ok := e.Entities[x]
			if !ok || !same(z, got) {
				return fmt.Errorf("project_v8.component:%s", x)
			}
			used[x] = true
		}
	}
	if len(used) != len(e.Entities) {
		return fmt.Errorf("project_v8.orphan")
	}
	return nil
}

// matchPackages binds the complete Package-v4 graph back to the package list
// and root in the same-run ProjectSnapshot. Package-v4's own validator proves
// each detail/member graph; this check prevents mixing two independently valid
// project/package closures.
func matchPackages(e wire.Envelope, snapshot, graph wire.ID) error {
	s := e.Entities[snapshot]
	pv, root := s.Fields[id("e112")], s.Fields[id("e113")]
	complete := e.Entities[graph]
	if complete.Schema != id("b029") || complete.Fields[id("b290")].Tag != 6 {
		return fmt.Errorf("project_v8.package_shape")
	}
	g := e.Entities[complete.Fields[id("b290")].Reference]
	dv := g.Fields[id("b200")]
	if g.Schema != id("b020") || pv.Tag != 7 || root.Tag != 6 || dv.Tag != 7 {
		return fmt.Errorf("project_v8.package_shape")
	}
	packages := map[wire.ID]bool{}
	for _, v := range pv.List {
		if v.Tag != 6 || packages[v.Reference] {
			return fmt.Errorf("project_v8.package_shape")
		}
		packages[v.Reference] = true
	}
	seen := map[wire.ID]bool{}
	adj := map[wire.ID][]wire.ID{}
	for _, v := range dv.List {
		if v.Tag != 6 {
			return fmt.Errorf("project_v8.package_detail")
		}
		d := e.Entities[v.Reference]
		if d.Schema != id("b021") || d.Fields[id("b210")].Tag != 6 {
			return fmt.Errorf("project_v8.package_detail")
		}
		p := d.Fields[id("b210")].Reference
		if !packages[p] || seen[p] {
			return fmt.Errorf("project_v8.package_coverage")
		}
		seen[p] = true
		for _, dep := range e.Entities[p].Fields[id("b103")].List {
			q := e.Entities[dep.Reference]
			if q.Fields[id("b122")].Tag == 6 {
				adj[p] = append(adj[p], q.Fields[id("b122")].Reference)
			}
		}
	}
	if len(seen) != len(packages) || !seen[root.Reference] {
		return fmt.Errorf("project_v8.package_coverage")
	}
	reachable := map[wire.ID]bool{}
	queue := []wire.ID{root.Reference}
	for len(queue) != 0 {
		x := queue[0]
		queue = queue[1:]
		if reachable[x] {
			continue
		}
		reachable[x] = true
		queue = append(queue, adj[x]...)
	}
	if len(reachable) != len(packages) {
		return fmt.Errorf("project_v8.package_reachability")
	}
	return nil
}

type rootSet struct{ inventory, dependency, packages, program, configuration wire.ID }

func components(in Inputs) ([]wire.Envelope, rootSet, error) {
	var roots rootSet
	if !in.Contracts.Validated() || in.Contracts.Foundation().Pin() != (contractcatalog.Pin{Module: id("3000"), Revision: id("3001")}) || in.Contracts.Execution().Pin() != (contractcatalog.Pin{Module: id("9000"), Revision: id("9024")}) || in.Contracts.Package().Pin() != (contractcatalog.Pin{Module: id("b000"), Revision: id("b004")}) || in.Contracts.Dependency().Pin() != (contractcatalog.Pin{Module: id("f000"), Revision: id("f001")}) || in.Contracts.Configuration().Pin() != (contractcatalog.Pin{Module: id("4000"), Revision: id("4006")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00a")}) {
		return nil, roots, fmt.Errorf("project_v8.contracts")
	}
	if err := sourceinventory.ValidateV8(in.Contracts.Project(), in.ProjectBase, in.Inventory); err != nil {
		return nil, roots, fmt.Errorf("project_v8.inventory:%w", err)
	}
	if err := packagev3instance.ValidateV4(in.Contracts, in.PackageV2, in.PackageV4); err != nil {
		return nil, roots, fmt.Errorf("project_v8.package:%w", err)
	}
	if !bytes.Equal(in.Configuration.Base.PackageV2, in.PackageV2) || !bytes.Equal(in.Configuration.Base.PackageV4, in.PackageV4) {
		return nil, roots, fmt.Errorf("project_v8.configuration_package_mismatch")
	}
	in.Configuration.Contracts = in.Contracts
	if err := configurationinstance.ValidateV3(in.Configuration); err != nil {
		return nil, roots, fmt.Errorf("project_v8.configuration:%w", err)
	}
	if _, err := dependencyinstance.Validate(in.Contracts.Dependency(), in.Dependency); err != nil {
		return nil, roots, fmt.Errorf("project_v8.dependency:%w", err)
	}
	construction, err := wire.Decode(in.Construction)
	if err != nil {
		return nil, roots, err
	}
	if err = executionprofile.ValidateConstruction(in.Contracts.Execution(), construction); err != nil {
		return nil, roots, err
	}
	inventory, _ := wire.Decode(in.Inventory)
	packages, _ := wire.Decode(in.PackageV4)
	dependency, _ := wire.Decode(in.Dependency)
	configuration, _ := wire.Decode(in.Configuration.Artifact)
	parts := []wire.Envelope{inventory, packages, dependency, construction, configuration}
	roots.inventory, err = one(inventory, id("e016"))
	if err != nil {
		return nil, roots, err
	}
	roots.dependency, err = one(dependency, id("f010"))
	if err != nil {
		return nil, roots, err
	}
	roots.packages, err = one(packages, id("b029"))
	if err != nil {
		return nil, roots, err
	}
	roots.program, err = one(construction, id("9015"))
	if err != nil {
		return nil, roots, err
	}
	roots.configuration, err = one(configuration, id("401f"))
	if err != nil {
		return nil, roots, err
	}
	if err = sameProgram(construction, packages, roots.program); err != nil {
		return nil, roots, err
	}
	return parts, roots, nil
}
func sameProgram(construction, packages wire.Envelope, root wire.ID) error {
	a, err := closure(construction.Entities, root)
	if err != nil {
		return err
	}
	b, err := closure(packages.Entities, root)
	if err != nil {
		return err
	}
	if len(a) != len(b) {
		return fmt.Errorf("project_v8.construction_mismatch")
	}
	for x, q := range a {
		if z, ok := b[x]; !ok || !same(q, z) {
			return fmt.Errorf("project_v8.construction_mismatch")
		}
	}
	return nil
}
func matchOrigins(e wire.Envelope, inventory, packageGraph wire.ID) error {
	units := map[wire.ID]wire.Entity{}
	for _, v := range e.Entities[inventory].Fields[id("e164")].List {
		if v.Tag != 6 {
			return fmt.Errorf("project_v8.inventory_units")
		}
		units[v.Reference] = e.Entities[v.Reference]
	}
	base := e.Entities[packageGraph].Fields[id("b290")].Reference
	owners := map[wire.ID]wire.ID{}
	used := map[wire.ID]bool{}
	for _, dv := range e.Entities[base].Fields[id("b200")].List {
		d := e.Entities[dv.Reference]
		pid := d.Fields[id("b210")].Reference
		for _, ov := range d.Fields[id("b213")].List {
			o := e.Entities[ov.Reference]
			uid := o.Fields[id("b260")].Reference
			u, ok := units[uid]
			if !ok || !bytes.Equal(u.Fields[id("e150")].Bytes, o.Fields[id("b261")].Bytes) || !bytes.Equal(u.Fields[id("e151")].Bytes, o.Fields[id("b262")].Bytes) {
				return fmt.Errorf("project_v8.origin_drift:%s", uid)
			}
			class := e.Entities[u.Fields[id("e153")].Reference]
			pres := e.Entities[u.Fields[id("e154")].Reference]
			if class.Fields[id("e130")].Unsigned != 0 || pres.Fields[id("e140")].Unsigned != 1 {
				return fmt.Errorf("project_v8.origin_classification:%s", uid)
			}
			if prior, yes := owners[uid]; yes && prior != pid {
				return fmt.Errorf("project_v8.source_owned_twice:%s", uid)
			}
			owners[uid] = pid
			used[uid] = true
		}
	}
	for uid, u := range units {
		class := e.Entities[u.Fields[id("e153")].Reference]
		pres := e.Entities[u.Fields[id("e154")].Reference]
		if class.Fields[id("e130")].Unsigned == 0 && pres.Fields[id("e140")].Unsigned == 1 && !used[uid] {
			return fmt.Errorf("project_v8.source_unowned:%s", uid)
		}
	}
	return nil
}
func pins(e wire.Envelope, m wire.Entity) bool {
	want := map[wire.ID]wire.ID{id("3000"): id("3001"), id("9000"): id("9024"), id("b000"): id("b004"), id("f000"): id("f001"), id("4000"): id("4006"), id("e000"): id("e00a")}
	v := m.Fields[id("121")]
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
		var rev wire.ID
		copy(rev[:], q.Fields[id("131")].Bytes)
		if _, ok := got[q.Fields[id("130")].Reference]; ok {
			return false
		}
		got[q.Fields[id("130")].Reference] = rev
	}
	for x, r := range want {
		if got[x] != r {
			return false
		}
	}
	return true
}
func SnapshotRevision(e wire.Envelope, root wire.ID) []byte {
	q := clone(e.Entities[root])
	q.Fields[id("e235")] = blob(make([]byte, sha256.Size))
	entities := map[wire.ID]wire.Entity{}
	for x, v := range e.Entities {
		entities[x] = v
	}
	entities[root] = q
	c, _ := closure(entities, root)
	b, _ := wire.Encode(wire.Envelope{Entities: c})
	h := sha256.Sum256(append([]byte("seme.full-configured-project.snapshot.v1\x00"), b...))
	return h[:]
}
func ArtifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.full-configured-project.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(in Inputs, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.project-v8.identity.v1\x00"))
	for _, raw := range [][]byte{in.Inventory, in.Dependency, in.PackageV4, in.Construction, in.Configuration.Artifact} {
		x := sha256.Sum256(raw)
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
func closure(es map[wire.ID]wire.Entity, root wire.ID) (map[wire.ID]wire.Entity, error) {
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
			return nil, fmt.Errorf("project_v8.reference_missing:%s", x)
		}
		out[x] = q
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return out, nil
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
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	for id, q := range e.Entities {
		if q.Schema == s {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_v8.schema_count:%s", s)
			}
			x = id
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_v8.schema_count:%s", s)
	}
	return x, nil
}
func same(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Module: a.ID, Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Module: b.ID, Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func clone(q wire.Entity) wire.Entity {
	f := map[wire.ID]wire.Value{}
	for x, v := range q.Fields {
		f[x] = v
	}
	q.Fields = f
	return q
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
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return bytes.Compare(x[i].Reference[:], x[j].Reference[:]) < 0 })
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
