// Package resourceinstance emits and validates bounded detached Resource-v1 manifests.
package resourceinstance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv8instance"
	"seme.local/reference/wire"
)

const MaxResources = 16
const MaxResourceBytes = 64 << 10
const MaxTotalBytes = 256 << 10

type Resource struct {
	Identity          string
	Owner, SourceUnit wire.ID
	Path              string
	Kind              uint64
	MediaType         string
	Size              uint64
	SHA256            [32]byte
}
type Placement struct{ Resource, Destination string }
type Model struct {
	Resources  []Resource
	Placements []Placement
}
type Inputs struct {
	Contracts contractcatalog.ProjectContractSetV9
	ProjectV8 projectv8instance.Inputs
	Model     Model
	Artifact  []byte
}

// ModelFromArtifact reconstructs the declarative model. Authenticity is not
// implied; callers must pass the returned model to Validate with exact inputs.
func ModelFromArtifact(artifact []byte) (Model, error) {
	e, err := wire.Decode(artifact)
	if err != nil {
		return Model{}, err
	}
	byID := map[wire.ID]string{}
	var out Model
	for x, q := range e.Entities {
		if q.Schema != id("6011") {
			continue
		}
		name, owner, source, pathv, kind, media, size, digest := q.Fields[id("6110")], q.Fields[id("6111")], q.Fields[id("6112")], q.Fields[id("6113")], q.Fields[id("6114")], q.Fields[id("6115")], q.Fields[id("6116")], q.Fields[id("6117")]
		ke, ok := e.Entities[kind.Reference]
		if name.Tag != 5 || owner.Tag != 6 || source.Tag != 6 || pathv.Tag != 5 || kind.Tag != 6 || !ok || ke.Schema != id("6012") || media.Tag != 5 || size.Tag != 3 || digest.Tag != 5 || len(digest.Bytes) != 32 {
			return Model{}, fmt.Errorf("resource_instance.decode")
		}
		var sum [32]byte
		copy(sum[:], digest.Bytes)
		n := string(name.Bytes)
		byID[x] = n
		out.Resources = append(out.Resources, Resource{Identity: n, Owner: owner.Reference, SourceUnit: source.Reference, Path: string(pathv.Bytes), Kind: ke.Fields[id("6120")].Unsigned, MediaType: string(media.Bytes), Size: size.Unsigned, SHA256: sum})
	}
	for _, q := range e.Entities {
		if q.Schema == id("6013") {
			r, d := q.Fields[id("6130")], q.Fields[id("6131")]
			n, ok := byID[r.Reference]
			if r.Tag != 6 || d.Tag != 5 || !ok {
				return Model{}, fmt.Errorf("resource_instance.decode_placement")
			}
			out.Placements = append(out.Placements, Placement{Resource: n, Destination: string(d.Bytes)})
		}
	}
	if len(out.Resources) == 0 {
		return Model{}, fmt.Errorf("resource_instance.decode_empty")
	}
	return out, nil
}

// VerifyDetached authenticates the external content store independently of
// canonical metadata. Callers remain responsible for race-safe filesystem or
// transport acquisition before constructing this immutable map.
func VerifyDetached(in Inputs, blobs map[[32]byte][]byte) error {
	if err := Validate(in); err != nil {
		return err
	}
	used := map[[32]byte]bool{}
	for _, resource := range in.Model.Resources {
		data, ok := blobs[resource.SHA256]
		if !ok || uint64(len(data)) != resource.Size || sha256.Sum256(data) != resource.SHA256 {
			return fmt.Errorf("resource_instance.detached:%s", resource.Identity)
		}
		if resource.Kind == 0 && !utf8.Valid(data) {
			return fmt.Errorf("resource_instance.text:%s", resource.Identity)
		}
		used[resource.SHA256] = true
	}
	if len(used) != len(blobs) {
		return fmt.Errorf("resource_instance.detached_orphan")
	}
	return nil
}

func Emit(in Inputs) ([]byte, error) { return emit(in) }
func Validate(in Inputs) error {
	if err := contracts(in); err != nil {
		return err
	}
	want, err := emit(Inputs{Contracts: in.Contracts, ProjectV8: in.ProjectV8, Model: in.Model})
	if err != nil {
		return err
	}
	if !bytes.Equal(want, in.Artifact) {
		return fmt.Errorf("resource_instance.artifact")
	}
	return nil
}

func emit(in Inputs) ([]byte, error) {
	if err := contracts(in); err != nil {
		return nil, err
	}
	base, err := wire.Decode(in.ProjectV8.Composed)
	if err != nil {
		return nil, err
	}
	resources := append([]Resource(nil), in.Model.Resources...)
	placements := append([]Placement(nil), in.Model.Placements...)
	sort.Slice(resources, func(i, j int) bool { return resources[i].Identity < resources[j].Identity })
	sort.Slice(placements, func(i, j int) bool {
		if placements[i].Destination == placements[j].Destination {
			return placements[i].Resource < placements[j].Resource
		}
		return placements[i].Destination < placements[j].Destination
	})
	if len(resources) == 0 || len(resources) > MaxResources || len(placements) != len(resources) {
		return nil, fmt.Errorf("resource_instance.count")
	}
	units := inventoryUnits(base)
	owners := packageOwners(base)
	by := map[string]wire.ID{}
	var total uint64
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for x, q := range base.Entities {
		if q.Schema != id("12") && q.Schema != id("13") {
			e.Entities[x] = q
		}
	}
	refs := []wire.Value{}
	for i, r := range resources {
		if !logical(r.Identity) || i > 0 && resources[i-1].Identity == r.Identity || !path(r.Path) || !media(r.MediaType) || r.Kind > 1 || r.Size > MaxResourceBytes || r.Size == 0 || !owners[r.Owner] {
			return nil, fmt.Errorf("resource_instance.resource:%s", r.Identity)
		}
		u, ok := units[r.SourceUnit]
		if !ok || string(u.Fields[id("e150")].Bytes) != r.Path || u.Fields[id("e152")].Unsigned != r.Size || !bytes.Equal(u.Fields[id("e151")].Bytes, r.SHA256[:]) || base.Entities[u.Fields[id("e154")].Reference].Fields[id("e140")].Unsigned != 0 {
			return nil, fmt.Errorf("resource_instance.source:%s", r.Identity)
		}
		// Text validity is checked by the detached byte resolver; blobs are never
		// embedded in this canonical metadata artifact.
		total += r.Size
		if total > MaxTotalBytes {
			return nil, fmt.Errorf("resource_instance.total")
		}
		x := stable(in.ProjectV8.Composed, "resource", r.Identity)
		k := stable(in.ProjectV8.Composed, "kind", fmt.Sprint(r.Kind))
		e.Entities[k] = wire.Entity{ID: k, Schema: id("6012"), Version: 1, Fields: map[wire.ID]wire.Value{id("6120"): uval(r.Kind)}}
		e.Entities[x] = wire.Entity{ID: x, Schema: id("6011"), Version: 1, Fields: map[wire.ID]wire.Value{id("6110"): blob([]byte(r.Identity)), id("6111"): ref(r.Owner), id("6112"): ref(r.SourceUnit), id("6113"): blob([]byte(r.Path)), id("6114"): ref(k), id("6115"): blob([]byte(r.MediaType)), id("6116"): uval(r.Size), id("6117"): blob(r.SHA256[:])}}
		by[r.Identity] = x
		refs = append(refs, ref(x))
	}
	prefs := []wire.Value{}
	destinations := []string{}
	seenPlacement := map[string]bool{}
	for _, p := range placements {
		r, ok := by[p.Resource]
		if !ok || seenPlacement[p.Resource] || !path(p.Destination) {
			return nil, fmt.Errorf("resource_instance.placement")
		}
		seenPlacement[p.Resource] = true
		for _, d := range destinations {
			if collide(d, p.Destination) {
				return nil, fmt.Errorf("resource_instance.destination")
			}
		}
		destinations = append(destinations, p.Destination)
		x := stable(in.ProjectV8.Composed, "placement", p.Resource, p.Destination)
		e.Entities[x] = wire.Entity{ID: x, Schema: id("6013"), Version: 1, Fields: map[wire.ID]wire.Value{id("6130"): ref(r), id("6131"): blob([]byte(p.Destination))}}
		prefs = append(prefs, ref(x))
	}
	sortRefs(refs)
	sortRefs(prefs)
	manifest := stable(in.ProjectV8.Composed, "manifest")
	e.Entities[manifest] = wire.Entity{ID: manifest, Schema: id("6010"), Version: 1, Fields: map[wire.ID]wire.Value{id("6100"): {Tag: 7, List: refs}, id("6101"): {Tag: 7, List: prefs}, id("6102"): blob(make([]byte, 32))}}
	q := e.Entities[manifest]
	q.Fields[id("6102")] = blob(revision(e, manifest, id("6102"), "seme.resource-manifest.v1\x00"))
	e.Entities[manifest] = q
	module := stable(in.ProjectV8.Composed, "module")
	imports := []wire.Value{}
	for _, pin := range []contractcatalog.Pin{{Module: id("b000"), Revision: id("b004")}, {Module: id("6000"), Revision: id("6001")}, {Module: id("e000"), Revision: id("e00a")}} {
		x := stable(in.ProjectV8.Composed, "import", pin.Module.String())
		e.Entities[x] = wire.Entity{ID: x, Schema: id("13"), Version: 1, Fields: map[wire.ID]wire.Value{id("130"): ref(pin.Module), id("131"): blob(pin.Revision[:])}}
		imports = append(imports, ref(x))
	}
	sortRefs(imports)
	e.Module = module
	e.Entities[module] = wire.Entity{ID: module, Schema: id("12"), Version: 1, Fields: map[wire.ID]wire.Value{id("120"): blob([]byte("resource-manifest-v1")), id("121"): {Tag: 7, List: imports}, id("122"): {Tag: 7, List: []wire.Value{ref(manifest)}}}}
	e.Revision = artifactRevision(e)
	return wire.Encode(e)
}

func contracts(in Inputs) error {
	if !in.Contracts.Validated() || in.Contracts.Resource().Pin() != (contractcatalog.Pin{Module: id("6000"), Revision: id("6001")}) || in.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00b")}) || in.Contracts.Package().Pin() != (contractcatalog.Pin{Module: id("b000"), Revision: id("b004")}) {
		return fmt.Errorf("resource_instance.contracts")
	}
	if !in.ProjectV8.Contracts.Validated() || in.ProjectV8.Contracts.Foundation().Pin() != in.Contracts.Foundation().Pin() || in.ProjectV8.Contracts.Execution().Pin() != in.Contracts.Execution().Pin() || in.ProjectV8.Contracts.Package().Pin() != in.Contracts.Package().Pin() || in.ProjectV8.Contracts.Dependency().Pin() != in.Contracts.Dependency().Pin() || in.ProjectV8.Contracts.Configuration().Pin() != in.Contracts.Configuration().Pin() || in.ProjectV8.Contracts.Project().Pin() != (contractcatalog.Pin{Module: id("e000"), Revision: id("e00a")}) {
		return fmt.Errorf("resource_instance.project_contracts")
	}
	if err := projectv8instance.Validate(in.ProjectV8); err != nil {
		return fmt.Errorf("resource_instance.project_v8:%w", err)
	}
	return nil
}
func inventoryUnits(e wire.Envelope) map[wire.ID]wire.Entity {
	out := map[wire.ID]wire.Entity{}
	for _, q := range e.Entities {
		if q.Schema == id("e016") {
			for _, r := range q.Fields[id("e164")].List {
				if u, ok := e.Entities[r.Reference]; ok && u.Schema == id("e015") {
					out[r.Reference] = u
				}
			}
		}
	}
	return out
}
func packageOwners(e wire.Envelope) map[wire.ID]bool {
	out := map[wire.ID]bool{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			out[x] = true
		}
	}
	return out
}
func logical(s string) bool {
	if len(s) == 0 || len(s) > 128 || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-' || r == '/') {
			return false
		}
	}
	return path(s)
}
func media(s string) bool {
	if len(s) < 3 || len(s) > 128 || strings.Count(s, "/") != 1 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("!#$&^_.+-/;= ", r)) {
			return false
		}
	}
	return !strings.HasPrefix(s, "/") && !strings.HasSuffix(s, "/")
}
func path(s string) bool {
	if len(s) == 0 || len(s) > 512 || strings.HasPrefix(s, "/") || strings.HasSuffix(s, "/") || strings.ContainsAny(s, "\\\x00") || !utf8.ValidString(s) {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func collide(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
func revision(e wire.Envelope, root, field wire.ID, domain string) []byte {
	q := e.Entities[root]
	q.Fields = cloneFields(q.Fields)
	q.Fields[field] = blob(make([]byte, 32))
	es := map[wire.ID]wire.Entity{}
	for x, v := range e.Entities {
		es[x] = v
	}
	es[root] = q
	c := closure(es, root)
	b, _ := wire.Encode(wire.Envelope{Entities: c})
	h := sha256.Sum256(append([]byte(domain), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.resource-artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
}
func stable(seed []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.resource-instance.identity.v1\x00"))
	s := sha256.Sum256(seed)
	h.Write(s[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
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
func collect(v wire.Value, x *[]wire.ID) {
	if v.Tag == 6 {
		*x = append(*x, v.Reference)
	}
	if v.Tag == 7 {
		for _, q := range v.List {
			collect(q, x)
		}
	}
}
func cloneFields(x map[wire.ID]wire.Value) map[wire.ID]wire.Value {
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
func uval(x uint64) wire.Value { return wire.Value{Tag: 3, Unsigned: x} }
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
