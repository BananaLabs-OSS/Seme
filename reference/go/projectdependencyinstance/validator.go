// Package projectdependencyinstance validates Project Contract v4 instances.
// It binds an independently authenticated Project v3 graph to an independently
// authenticated Dependency v1 closure without introducing provider source.
package projectdependencyinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/wire"
)

var (
	moduleSchema   = id("12")
	importSchema   = id("13")
	graphSchema    = id("e018")
	closureSchema  = id("f010")
	snapshotSchema = id("e019")
)

type Inputs struct {
	Contracts  contractcatalog.ProjectContractSetV4
	ProjectV3  projectgraphinstance.Inputs
	Dependency []byte
	Composed   []byte
}

var requiredPins = map[wire.ID]wire.ID{id("9000"): id("9023"), id("b000"): id("b002"), id("f000"): id("f001"), id("e000"): id("e004")}

func Validate(in Inputs) error {
	if !in.Contracts.Validated() || in.Contracts.Execution().Pin() != (contractcatalog.Pin{Module: id("9000"), Revision: id("9023")}) || in.Contracts.Package().Pin() != (contractcatalog.Pin{Module: id("b000"), Revision: id("b002")}) || in.Contracts.Dependency().Pin() != (contractcatalog.Pin{Module: id("f000"), Revision: id("f001")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e004")}) {
		return fmt.Errorf("project_dependency.contracts")
	}
	if err := projectgraphinstance.Validate(in.ProjectV3); err != nil {
		return fmt.Errorf("project_dependency.project_v3:%w", err)
	}
	if _, err := dependencyinstance.Validate(in.Contracts.Dependency(), in.Dependency); err != nil {
		return fmt.Errorf("project_dependency.dependency:%w", err)
	}
	p, _ := wire.Decode(in.ProjectV3.Composed)
	d, _ := wire.Decode(in.Dependency)
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return fmt.Errorf("project_dependency.wire:%w", err)
	}
	canonical, err := wire.Encode(e)
	if err != nil || !bytes.Equal(canonical, in.Composed) {
		return fmt.Errorf("project_dependency.noncanonical")
	}
	if err = validateModule(e); err != nil {
		return err
	}
	if len(e.Parents) != 0 || e.Module != Identity(in.ProjectV3.Composed, in.Dependency, "module") {
		return fmt.Errorf("project_dependency.envelope")
	}
	for _, v := range e.Entities[e.Module].Fields[id("121")].List {
		q := e.Entities[v.Reference]
		if v.Reference != Identity(in.ProjectV3.Composed, in.Dependency, "import:"+q.Fields[id("130")].Reference.String()) {
			return fmt.Errorf("project_dependency.import_identity")
		}
	}
	if revision, er := ArtifactRevision(e); er != nil || revision != e.Revision {
		return fmt.Errorf("project_dependency.artifact_revision")
	}
	graph, err := one(p, graphSchema)
	if err != nil {
		return err
	}
	closure, err := one(d, closureSchema)
	if err != nil {
		return err
	}
	snapshot, err := one(e, snapshotSchema)
	if err != nil {
		return err
	}
	if snapshot != Identity(in.ProjectV3.Composed, in.Dependency, "snapshot") {
		return fmt.Errorf("project_dependency.snapshot_identity")
	}
	s := e.Entities[snapshot]
	if !shape(s, []wire.ID{id("e190"), id("e191"), id("e192")}, []byte{6, 6, 5}) || s.Fields[id("e190")].Reference != graph || s.Fields[id("e191")].Reference != closure || len(s.Fields[id("e192")].Bytes) != sha256.Size {
		return fmt.Errorf("project_dependency.snapshot")
	}
	if got, er := SnapshotRevision(e, snapshot); er != nil || !bytes.Equal(got, s.Fields[id("e192")].Bytes) {
		return fmt.Errorf("project_dependency.snapshot_revision")
	}
	return exactUnion(e, p, d, snapshot)
}

// Identity derives an instance entity identity solely from the two validated
// canonical input artifacts and a role. It is exported so the emitter and
// validator cannot drift onto different identity algorithms.
func Identity(projectV3, dependency []byte, role string) wire.ID {
	key := sha256.New()
	key.Write([]byte("seme.project-v4-emitter.v1\x00"))
	for _, raw := range [][]byte{projectV3, dependency} {
		h := sha256.Sum256(raw)
		key.Write(h[:])
	}
	h := sha256.New()
	h.Write([]byte("seme.project-v4-emitter.identity.v1\x00"))
	k := key.Sum(nil)
	_ = binary.Write(h, binary.BigEndian, uint64(len(k)))
	h.Write(k)
	_ = binary.Write(h, binary.BigEndian, uint64(len(role)))
	h.Write([]byte(role))
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}

func SnapshotRevision(e wire.Envelope, snapshot wire.ID) ([]byte, error) {
	return revision(e, snapshot, id("e192"), "seme.dependency-graph-snapshot.v1\x00")
}
func ArtifactRevision(e wire.Envelope) (wire.ID, error) {
	e.Revision = wire.ID{}
	b, err := wire.Encode(e)
	if err != nil {
		return wire.ID{}, err
	}
	h := sha256.Sum256(append([]byte("seme.project-dependency.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x, nil
}

func validateModule(e wire.Envelope) error {
	m, ok := e.Entities[e.Module]
	if !ok || m.Schema != moduleSchema || m.Version != 1 || !shape(m, []wire.ID{id("120"), id("121"), id("122")}, []byte{5, 7, 7}) || !bytes.Equal(m.Fields[id("120")].Bytes, []byte("project-dependency-graph-v1")) {
		return fmt.Errorf("project_dependency.module")
	}
	imports := m.Fields[id("121")].List
	if len(imports) != 4 {
		return fmt.Errorf("project_dependency.import_count")
	}
	pins, seen := map[wire.ID]wire.ID{}, map[wire.ID]bool{}
	var prior wire.ID
	for i, v := range imports {
		if v.Tag != 6 || seen[v.Reference] || (i > 0 && !less(prior, v.Reference)) {
			return fmt.Errorf("project_dependency.imports")
		}
		seen[v.Reference], prior = true, v.Reference
		q, yes := e.Entities[v.Reference]
		if !yes || !shape(q, []wire.ID{id("130"), id("131")}, []byte{6, 5}) || q.Schema != importSchema || len(q.Fields[id("131")].Bytes) != 16 {
			return fmt.Errorf("project_dependency.import_shape")
		}
		var r wire.ID
		copy(r[:], q.Fields[id("131")].Bytes)
		mod := q.Fields[id("130")].Reference
		if _, dup := pins[mod]; dup {
			return fmt.Errorf("project_dependency.import_duplicate")
		}
		pins[mod] = r
	}
	if len(pins) != len(requiredPins) {
		return fmt.Errorf("project_dependency.import_pin")
	}
	for m, r := range requiredPins {
		if pins[m] != r {
			return fmt.Errorf("project_dependency.import_pin")
		}
	}
	for x, q := range e.Entities {
		if q.Schema == importSchema && !seen[x] {
			return fmt.Errorf("project_dependency.orphan_import:%s", x)
		}
	}
	exports := m.Fields[id("122")].List
	if len(exports) != 1 || exports[0].Tag != 6 || e.Entities[exports[0].Reference].Schema != snapshotSchema {
		return fmt.Errorf("project_dependency.exports")
	}
	return nil
}

func exactUnion(out, p, d wire.Envelope, snapshot wire.ID) error {
	want := map[wire.ID]wire.Entity{}
	for _, src := range []wire.Envelope{p, d} {
		for x, q := range src.Entities {
			if q.Schema == moduleSchema || q.Schema == importSchema {
				continue
			}
			if _, exists := want[x]; exists {
				return fmt.Errorf("project_dependency.component_collision:%s", x)
			}
			want[x] = q
		}
	}
	want[snapshot] = out.Entities[snapshot]
	for x, q := range want {
		if got, ok := out.Entities[x]; !ok || !equal(got, q) {
			return fmt.Errorf("project_dependency.component_drift:%s", x)
		}
	}
	for x, q := range out.Entities {
		if x == out.Module || q.Schema == importSchema {
			continue
		}
		if _, ok := want[x]; !ok {
			return fmt.Errorf("project_dependency.orphan:%s", x)
		}
	}
	return nil
}

func revision(e wire.Envelope, root, exclude wire.ID, domain string) ([]byte, error) {
	seen := map[wire.ID]bool{}
	queue := []wire.ID{root}
	out := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for len(queue) > 0 {
		x := queue[0]
		queue = queue[1:]
		if seen[x] {
			continue
		}
		q, ok := e.Entities[x]
		if !ok {
			return nil, fmt.Errorf("project_dependency.revision_missing:%s", x)
		}
		seen[x] = true
		fields := map[wire.ID]wire.Value{}
		for f, v := range q.Fields {
			if x == root && f == exclude {
				continue
			}
			fields[f] = v
			collect(v, &queue)
		}
		q.Fields = fields
		out.Entities[x] = q
	}
	b, err := wire.Encode(out)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write(b)
	return h.Sum(nil), nil
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
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	n := 0
	for id, q := range e.Entities {
		if q.Schema == s {
			x = id
			n++
		}
	}
	if n != 1 {
		return x, fmt.Errorf("project_dependency.schema_count:%s:%d", s, n)
	}
	return x, nil
}
func shape(e wire.Entity, fs []wire.ID, tags []byte) bool {
	if e.Version != 1 || len(e.Fields) != len(fs) {
		return false
	}
	for i, f := range fs {
		v, ok := e.Fields[f]
		if !ok || v.Tag != tags[i] {
			return false
		}
	}
	return true
}
func equal(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func less(a, b wire.ID) bool { return bytes.Compare(a[:], b[:]) < 0 }
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
