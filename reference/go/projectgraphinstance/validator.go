// Package projectgraphinstance validates the composed Project Contract v3
// trust boundary. It joins, but does not replace, the independently validated
// semantic Project, source inventory, and Package v2 detail artifacts.
package projectgraphinstance

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/projectinstance"
	"seme.local/reference/sourceinventory"
	"seme.local/reference/wire"
)

var (
	moduleSchema         = id("12")
	importSchema         = id("13")
	projectSchema        = id("e011")
	inventorySchema      = id("e016")
	bindingSchema        = id("e017")
	graphSnapshotSchema  = id("e018")
	packageGraphSchema   = id("b020")
	detailSchema         = id("b021")
	originSchema         = id("b026")
	classificationSchema = id("e013")
	preservationSchema   = id("e014")
)

type Inputs struct {
	// Contracts is an authenticated Project v3 contract set. Instance-declared
	// imports are checked independently and cannot substitute for this trust.
	Contracts contractcatalog.ProjectContractSet
	// ProjectV2 is the authenticated Project v2 contract used by the detached
	// source-inventory validator. The composed envelope itself pins Project v3.
	ProjectV2                                  contractcatalog.Contract
	Project, Inventory, PackageGraph, Composed []byte
}

var requiredPins = map[wire.ID]wire.ID{
	id("9000"): id("9023"), id("b000"): id("b002"), id("e000"): id("e003"),
}

func Validate(in Inputs) error {
	if err := validateContracts(in.Contracts); err != nil {
		return err
	}
	if err := projectinstance.Validate(in.Project); err != nil {
		return fmt.Errorf("project_graph.project:%w", err)
	}
	if err := sourceinventory.Validate(in.ProjectV2, in.Project, in.Inventory); err != nil {
		return fmt.Errorf("project_graph.inventory:%w", err)
	}
	if err := packagedetailinstance.Validate(in.PackageGraph); err != nil {
		return fmt.Errorf("project_graph.package:%w", err)
	}
	project, _ := wire.Decode(in.Project)
	inventory, _ := wire.Decode(in.Inventory)
	packages, _ := wire.Decode(in.PackageGraph)
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_graph.wire:%w", err)
	}
	canonical, err := wire.Encode(e)
	if err != nil || !bytes.Equal(canonical, in.Composed) {
		return fmt.Errorf("project_graph.noncanonical")
	}
	if err = validateModule(e); err != nil {
		return err
	}
	wantArtifact, err := ArtifactRevision(e)
	if err != nil || wantArtifact != e.Revision {
		return fmt.Errorf("project_graph.artifact_revision")
	}

	projectRoot, err := one(project, projectSchema)
	if err != nil {
		return err
	}
	inventoryRoot, err := one(inventory, inventorySchema)
	if err != nil {
		return err
	}
	packageRoot, err := one(packages, packageGraphSchema)
	if err != nil {
		return err
	}
	packageProjectRoot, err := one(packages, projectSchema)
	if err != nil || packageProjectRoot != projectRoot {
		return fmt.Errorf("project_graph.package_snapshot_binding")
	}
	binding, err := one(e, bindingSchema)
	if err != nil {
		return err
	}
	snapshot, err := one(e, graphSnapshotSchema)
	if err != nil {
		return err
	}
	b := e.Entities[binding]
	if !shape(b, []wire.ID{id("e170"), id("e171"), id("e172")}, []byte{6, 6, 5}) || b.Fields[id("e170")].Reference != packageRoot || b.Fields[id("e171")].Reference != inventoryRoot || len(b.Fields[id("e172")].Bytes) != sha256.Size {
		return fmt.Errorf("project_graph.binding")
	}
	s := e.Entities[snapshot]
	if !shape(s, []wire.ID{id("e180"), id("e181"), id("e182")}, []byte{6, 6, 5}) || s.Fields[id("e180")].Reference != projectRoot || s.Fields[id("e181")].Reference != binding || len(s.Fields[id("e182")].Bytes) != sha256.Size {
		return fmt.Errorf("project_graph.snapshot")
	}
	if got, er := BindingRevision(e, binding); er != nil || !bytes.Equal(got, b.Fields[id("e172")].Bytes) {
		return fmt.Errorf("project_graph.binding_revision")
	}
	if got, er := SnapshotRevision(e, snapshot); er != nil || !bytes.Equal(got, s.Fields[id("e182")].Bytes) {
		return fmt.Errorf("project_graph.snapshot_revision")
	}

	if err = exactUnion(e, []wire.Envelope{project, inventory, packages}, binding, snapshot); err != nil {
		return err
	}
	if err = matchPackages(e, projectRoot, packageRoot); err != nil {
		return err
	}
	if err = matchOrigins(e, inventoryRoot, packageRoot); err != nil {
		return err
	}
	return nil
}

func validateContracts(contracts contractcatalog.ProjectContractSet) error {
	if !contracts.Validated() {
		return fmt.Errorf("project_graph.contracts_unvalidated")
	}
	want := []contractcatalog.Pin{
		{Module: id("9000"), Revision: id("9023")},
		{Module: id("b000"), Revision: id("b002")},
		{Module: id("e000"), Revision: id("e003")},
	}
	got := []contractcatalog.Pin{contracts.Execution().Pin(), contracts.Package().Pin(), contracts.Project().Pin()}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("project_graph.contract_pin")
		}
	}
	return nil
}

func BindingRevision(e wire.Envelope, binding wire.ID) ([]byte, error) {
	return revision(e, binding, id("e172"), "seme.project-package-graph-binding.v1\x00")
}
func SnapshotRevision(e wire.Envelope, snapshot wire.ID) ([]byte, error) {
	return revision(e, snapshot, id("e182"), "seme.project-graph-snapshot.v1\x00")
}
func ArtifactRevision(e wire.Envelope) (wire.ID, error) {
	e.Revision = wire.ID{}
	raw, err := wire.Encode(e)
	if err != nil {
		return wire.ID{}, err
	}
	sum := sha256.Sum256(append([]byte("seme.project-graph.artifact.v1\x00"), raw...))
	var out wire.ID
	copy(out[:], sum[:16])
	return out, nil
}

func validateModule(e wire.Envelope) error {
	m, ok := e.Entities[e.Module]
	if !ok || m.Schema != moduleSchema || m.Version != 1 || len(m.Fields) != 3 || m.Fields[id("120")].Tag != 5 || m.Fields[id("121")].Tag != 7 || m.Fields[id("122")].Tag != 7 {
		return fmt.Errorf("project_graph.module")
	}
	imports := m.Fields[id("121")].List
	if len(imports) != 3 {
		return fmt.Errorf("project_graph.import_count")
	}
	seen := map[wire.ID]bool{}
	pins := map[wire.ID]wire.ID{}
	var prior wire.ID
	for i, v := range imports {
		if v.Tag != 6 || seen[v.Reference] || (i > 0 && !less(prior, v.Reference)) {
			return fmt.Errorf("project_graph.imports")
		}
		seen[v.Reference] = true
		prior = v.Reference
		q, yes := e.Entities[v.Reference]
		if !yes || q.Schema != importSchema || q.Version != 1 || len(q.Fields) != 2 || q.Fields[id("130")].Tag != 6 || q.Fields[id("131")].Tag != 5 || len(q.Fields[id("131")].Bytes) != 16 {
			return fmt.Errorf("project_graph.import_shape")
		}
		var r wire.ID
		copy(r[:], q.Fields[id("131")].Bytes)
		if _, dup := pins[q.Fields[id("130")].Reference]; dup {
			return fmt.Errorf("project_graph.import_duplicate")
		}
		pins[q.Fields[id("130")].Reference] = r
	}
	if len(pins) != len(requiredPins) {
		return fmt.Errorf("project_graph.import_pin")
	}
	for mod, rev := range requiredPins {
		if pins[mod] != rev {
			return fmt.Errorf("project_graph.import_pin")
		}
	}
	for x, q := range e.Entities {
		if q.Schema == importSchema && !seen[x] {
			return fmt.Errorf("project_graph.orphan_import:%s", x)
		}
	}
	exports := m.Fields[id("122")].List
	if len(exports) != 2 {
		return fmt.Errorf("project_graph.exports")
	}
	seenExport := map[wire.ID]bool{}
	seenSchema := map[wire.ID]bool{}
	var priorExport wire.ID
	for _, v := range exports {
		if v.Tag != 6 || seenExport[v.Reference] || (len(seenExport) > 0 && !less(priorExport, v.Reference)) {
			return fmt.Errorf("project_graph.exports")
		}
		q, ok := e.Entities[v.Reference]
		if !ok || (q.Schema != bindingSchema && q.Schema != graphSnapshotSchema) || seenSchema[q.Schema] {
			return fmt.Errorf("project_graph.exports")
		}
		seenExport[v.Reference], seenSchema[q.Schema], priorExport = true, true, v.Reference
	}
	if !seenSchema[bindingSchema] || !seenSchema[graphSnapshotSchema] {
		return fmt.Errorf("project_graph.exports")
	}
	return nil
}

func exactUnion(composed wire.Envelope, sources []wire.Envelope, binding, snapshot wire.ID) error {
	want := map[wire.ID]wire.Entity{}
	for _, src := range sources {
		for x, q := range src.Entities {
			if q.Schema == moduleSchema || q.Schema == importSchema {
				continue
			}
			if old, ok := want[x]; ok && !entityEqual(old, q) {
				return fmt.Errorf("project_graph.component_collision:%s", x)
			}
			want[x] = q
		}
	}
	want[binding] = composed.Entities[binding]
	want[snapshot] = composed.Entities[snapshot]
	for x, q := range want {
		got, ok := composed.Entities[x]
		if !ok || !entityEqual(got, q) {
			return fmt.Errorf("project_graph.component_drift:%s", x)
		}
	}
	for x, q := range composed.Entities {
		if x == composed.Module || q.Schema == importSchema {
			continue
		}
		if _, ok := want[x]; !ok {
			return fmt.Errorf("project_graph.orphan:%s", x)
		}
	}
	return nil
}

func matchPackages(e wire.Envelope, snapshot, graph wire.ID) error {
	s := e.Entities[snapshot]
	pv := s.Fields[id("e112")]
	root := s.Fields[id("e113")]
	g := e.Entities[graph]
	dv := g.Fields[id("b200")]
	if pv.Tag != 7 || root.Tag != 6 || dv.Tag != 7 {
		return fmt.Errorf("project_graph.package_shape")
	}
	packages := map[wire.ID]bool{}
	for _, v := range pv.List {
		if v.Tag != 6 {
			return fmt.Errorf("project_graph.package_shape")
		}
		packages[v.Reference] = true
	}
	seen := map[wire.ID]bool{}
	adj := map[wire.ID][]wire.ID{}
	for _, v := range dv.List {
		if v.Tag != 6 {
			return fmt.Errorf("project_graph.detail")
		}
		d := e.Entities[v.Reference]
		if d.Schema != detailSchema || d.Fields[id("b210")].Tag != 6 {
			return fmt.Errorf("project_graph.detail")
		}
		p := d.Fields[id("b210")].Reference
		if !packages[p] || seen[p] {
			return fmt.Errorf("project_graph.package_coverage")
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
		return fmt.Errorf("project_graph.package_coverage")
	}
	reachable := map[wire.ID]bool{}
	queue := []wire.ID{root.Reference}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if reachable[x] {
			continue
		}
		reachable[x] = true
		queue = append(queue, adj[x]...)
	}
	if len(reachable) != len(packages) {
		return fmt.Errorf("project_graph.package_reachability")
	}
	return nil
}

func matchOrigins(e wire.Envelope, inventory, graph wire.ID) error {
	units := map[wire.ID]wire.Entity{}
	inv := e.Entities[inventory]
	for _, v := range inv.Fields[id("e164")].List {
		if v.Tag != 6 {
			return fmt.Errorf("project_graph.inventory_units")
		}
		units[v.Reference] = e.Entities[v.Reference]
	}
	owners := map[wire.ID]wire.ID{}
	used := map[wire.ID]bool{}
	for _, dv := range e.Entities[graph].Fields[id("b200")].List {
		d := e.Entities[dv.Reference]
		pid := d.Fields[id("b210")].Reference
		for _, ov := range d.Fields[id("b213")].List {
			o := e.Entities[ov.Reference]
			if o.Schema != originSchema {
				return fmt.Errorf("project_graph.origin")
			}
			uid := o.Fields[id("b260")].Reference
			u, ok := units[uid]
			if !ok {
				return fmt.Errorf("project_graph.origin_source:%s", uid)
			}
			if u.Fields[id("e150")].Tag != 5 || u.Fields[id("e151")].Tag != 5 || !bytes.Equal(u.Fields[id("e150")].Bytes, o.Fields[id("b261")].Bytes) || !bytes.Equal(u.Fields[id("e151")].Bytes, o.Fields[id("b262")].Bytes) {
				return fmt.Errorf("project_graph.origin_drift:%s", uid)
			}
			class := e.Entities[u.Fields[id("e153")].Reference]
			pres := e.Entities[u.Fields[id("e154")].Reference]
			if class.Schema != classificationSchema || class.Fields[id("e130")].Tag != 3 || class.Fields[id("e130")].Unsigned != 0 || pres.Schema != preservationSchema || pres.Fields[id("e140")].Tag != 3 || pres.Fields[id("e140")].Unsigned != 1 {
				return fmt.Errorf("project_graph.origin_classification:%s", uid)
			}
			if prior, yes := owners[uid]; yes && prior != pid {
				return fmt.Errorf("project_graph.source_owned_twice:%s", uid)
			}
			owners[uid] = pid
			used[uid] = true
		}
	}
	// Every tracked semantic-projection unit represents executable semantic
	// source and therefore must be assigned to exactly one package.
	for uid, u := range units {
		class := e.Entities[u.Fields[id("e153")].Reference]
		pres := e.Entities[u.Fields[id("e154")].Reference]
		if class.Fields[id("e130")].Unsigned == 0 && pres.Fields[id("e140")].Unsigned == 1 && !used[uid] {
			return fmt.Errorf("project_graph.source_unowned:%s", uid)
		}
	}
	return nil
}

func revision(e wire.Envelope, root, excluded wire.ID, domain string) ([]byte, error) {
	seen := map[wire.ID]bool{}
	q := []wire.ID{root}
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if seen[x] {
			continue
		}
		v, ok := e.Entities[x]
		if !ok {
			return nil, fmt.Errorf("project_graph.revision_missing:%s", x)
		}
		seen[x] = true
		fields := map[wire.ID]wire.Value{}
		for f, z := range v.Fields {
			if x == root && f == excluded {
				continue
			}
			fields[f] = z
			collect(z, &q)
		}
		v.Fields = fields
		out.Entities[x] = v
	}
	raw, err := wire.Encode(out)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write(raw)
	return h.Sum(nil), nil
}
func one(e wire.Envelope, schema wire.ID) (wire.ID, error) {
	var x wire.ID
	for id, q := range e.Entities {
		if q.Schema == schema {
			if x != (wire.ID{}) {
				return x, fmt.Errorf("project_graph.schema_count:%s", schema)
			}
			x = id
		}
	}
	if x == (wire.ID{}) {
		return x, fmt.Errorf("project_graph.schema_count:%s", schema)
	}
	return x, nil
}
func shape(e wire.Entity, fields []wire.ID, tags []byte) bool {
	if e.Version != 1 || len(e.Fields) != len(fields) {
		return false
	}
	for i, f := range fields {
		v, ok := e.Fields[f]
		if !ok || v.Tag != tags[i] {
			return false
		}
	}
	return true
}
func collect(v wire.Value, q *[]wire.ID) {
	if v.Tag == 6 {
		*q = append(*q, v.Reference)
	}
	for _, x := range v.List {
		collect(x, q)
	}
	for _, x := range v.Record {
		collect(x, q)
	}
}
func entityEqual(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func less(a, b wire.ID) bool { return bytes.Compare(a[:], b[:]) < 0 }
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
