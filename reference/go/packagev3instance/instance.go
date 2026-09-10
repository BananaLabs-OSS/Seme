// Package packagev3instance emits and validates complete Package v3 ownership
// over an independently validated Package v2 graph.
package packagev3instance

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/wire"
)

type Kind uint64

const (
	DataType Kind = iota
	BehavioralInterface
	ReceiverCallable
	GenericDefinition
	GenericRealization
)

type Declaration struct {
	Identity, OwnerDetail string
	Origin                packagedetail.Origin
	Visibility            packagedetail.Visibility
	Kind                  Kind
	Name, ExportName      string
	ReferencedImports     []string
	GenericDefinition     string
}
type Inputs struct {
	Contracts    contractcatalog.ProjectContractSetV5
	PackageV2    []byte
	Declarations []Declaration
}

var relevant = map[wire.ID]Kind{id("9030"): DataType, id("a010"): BehavioralInterface, id("a002"): ReceiverCallable, id("9042"): GenericRealization, id("a004"): GenericRealization, id("a050"): GenericRealization}

func Emit(in Inputs) ([]byte, error) {
	base, err := validatedBase(in)
	if err != nil {
		return nil, err
	}
	e := cloneEnvelope(base)
	if err = upgrade(&e); err != nil {
		return nil, err
	}
	detailSet := schemaSet(e, id("b021"))
	bindingSet := schemaSet(e, id("b024"))
	kindIDs := map[Kind]wire.ID{}
	refs := []wire.Value{}
	for _, d := range in.Declarations {
		decl, er := wire.ParseID(d.Identity)
		if er != nil {
			return nil, fmt.Errorf("package_v3.declaration_identity")
		}
		owner, er := wire.ParseID(d.OwnerDetail)
		if er != nil || !detailSet[owner] {
			return nil, fmt.Errorf("package_v3.owner")
		}
		source, er := wire.ParseID(d.Origin.SourceIdentity)
		if er != nil || d.Origin.Path == "" || d.Origin.ByteStart >= d.Origin.ByteEnd || d.Origin.StartLine == 0 || d.Origin.StartColumn == 0 || d.Origin.EndLine == 0 || d.Origin.EndColumn == 0 || d.Origin.EndLine < d.Origin.StartLine {
			return nil, fmt.Errorf("package_v3.origin")
		}
		origin := stable(in.PackageV2, "origin", d.Origin.SourceIdentity, d.Origin.Path, fmt.Sprint(d.Origin.ByteStart), fmt.Sprint(d.Origin.ByteEnd))
		originEntity := wire.Entity{ID: origin, Schema: id("b026"), Version: 1, Fields: map[wire.ID]wire.Value{id("b260"): ref(source), id("b261"): blob([]byte(d.Origin.Path)), id("b262"): blob(d.Origin.ContentDigest[:]), id("b263"): u(d.Origin.ByteStart), id("b264"): u(d.Origin.ByteEnd), id("b265"): u(uint64(d.Origin.StartLine)), id("b266"): u(uint64(d.Origin.StartColumn)), id("b267"): u(uint64(d.Origin.EndLine)), id("b268"): u(uint64(d.Origin.EndColumn))}}
		for x, q := range e.Entities {
			if q.Schema == id("b026") && sameOrigin(q, originEntity) {
				origin = x
				originEntity.ID = x
				break
			}
		}
		if prior, exists := e.Entities[origin]; exists && !sameEntity(prior, originEntity) {
			return nil, fmt.Errorf("package_v3.identity_collision")
		}
		e.Entities[origin] = originEntity
		if d.Kind > GenericRealization {
			return nil, fmt.Errorf("package_v3.kind")
		}
		kid, ok := kindIDs[d.Kind]
		if !ok {
			kid = stable(in.PackageV2, "kind", fmt.Sprint(d.Kind))
			kindEntity := wire.Entity{ID: kid, Schema: id("b027"), Version: 1, Fields: map[wire.ID]wire.Value{id("b270"): u(uint64(d.Kind))}}
			if prior, exists := e.Entities[kid]; exists && !sameEntity(prior, kindEntity) {
				return nil, fmt.Errorf("package_v3.identity_collision")
			}
			e.Entities[kid] = kindEntity
			kindIDs[d.Kind] = kid
		}
		imports := []wire.Value{}
		for _, raw := range d.ReferencedImports {
			x, er := wire.ParseID(raw)
			if er != nil || !bindingSet[x] {
				return nil, fmt.Errorf("package_v3.referenced_import")
			}
			imports = append(imports, ref(x))
		}
		sortRefs(imports)
		if !unique(imports) {
			return nil, fmt.Errorf("package_v3.referenced_import_duplicate")
		}
		did := stable(in.PackageV2, "declaration", d.Identity)
		if d.Visibility > packagedetail.Public {
			return nil, fmt.Errorf("package_v3.visibility")
		}
		vid, er := visibility(e, in.PackageV2, uint64(d.Visibility))
		if er != nil {
			return nil, er
		}
		fs := map[wire.ID]wire.Value{id("b280"): ref(decl), id("b281"): ref(owner), id("b282"): ref(kid), id("b283"): blob([]byte(d.Name)), id("b284"): ref(vid), id("b286"): ref(origin), id("b287"): {Tag: 7, List: imports}}
		if d.ExportName != "" {
			fs[id("b285")] = blob([]byte(d.ExportName))
		}
		if d.GenericDefinition != "" {
			x, er := wire.ParseID(d.GenericDefinition)
			if er != nil {
				return nil, fmt.Errorf("package_v3.generic_definition")
			}
			fs[id("b288")] = ref(x)
		}
		if _, exists := e.Entities[did]; exists {
			return nil, fmt.Errorf("package_v3.identity_collision")
		}
		e.Entities[did] = wire.Entity{ID: did, Schema: id("b028"), Version: 1, Fields: fs}
		refs = append(refs, ref(did))
	}
	sortRefs(refs)
	gid := stable(in.PackageV2, "graph")
	if _, exists := e.Entities[gid]; exists {
		return nil, fmt.Errorf("package_v3.identity_collision")
	}
	baseGraph := one(e, id("b020"))
	e.Entities[gid] = wire.Entity{ID: gid, Schema: id("b029"), Version: 1, Fields: map[wire.ID]wire.Value{id("b290"): ref(baseGraph), id("b291"): {Tag: 7, List: refs}, id("b292"): blob(make([]byte, 32))}}
	m := e.Entities[e.Module]
	exports := append([]wire.Value(nil), m.Fields[id("122")].List...)
	exports = append(exports, ref(gid))
	sortRefs(exports)
	m.Fields[id("122")] = wire.Value{Tag: 7, List: exports}
	e.Entities[e.Module] = m
	q := e.Entities[gid]
	q.Fields[id("b292")] = blob(snapshotRevision(e, gid))
	e.Entities[gid] = q
	e.Revision = artifactRevision(e)
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	if err = Validate(in.Contracts, in.PackageV2, out); err != nil {
		return nil, fmt.Errorf("package_v3.emit_validate:%w", err)
	}
	return out, nil
}

func Validate(contracts contractcatalog.ProjectContractSetV5, baseRaw, outRaw []byte) error {
	if !contracts.Validated() || contracts.Package().Pin() != (contractcatalog.Pin{Module: id("b000"), Revision: id("b003")}) {
		return fmt.Errorf("package_v3.contracts")
	}
	if err := packagedetailinstance.Validate(baseRaw); err != nil {
		return fmt.Errorf("package_v3.base:%w", err)
	}
	base, _ := wire.Decode(baseRaw)
	e, err := wire.Decode(outRaw)
	if err != nil {
		return fmt.Errorf("package_v3.wire:%w", err)
	}
	canonical, _ := wire.Encode(e)
	if !bytes.Equal(canonical, outRaw) {
		return fmt.Errorf("package_v3.noncanonical")
	}
	if err = checkUpgrade(e); err != nil {
		return err
	}
	if e.Revision != artifactRevision(e) {
		return fmt.Errorf("package_v3.artifact_revision")
	}
	graphs := withSchema(e, id("b029"))
	if len(graphs) != 1 {
		return fmt.Errorf("package_v3.graph_count")
	}
	g := e.Entities[graphs[0]]
	if g.Version != 1 || !shape(g, []wire.ID{id("b290"), id("b291"), id("b292")}, []byte{6, 7, 5}) || g.Fields[id("b290")].Reference != one(base, id("b020")) {
		return fmt.Errorf("package_v3.graph")
	}
	m := e.Entities[e.Module]
	exports := m.Fields[id("122")].List
	if !uniqueSorted(exports) {
		return fmt.Errorf("package_v3.exports")
	}
	wantExports := append([]wire.Value(nil), base.Entities[base.Module].Fields[id("122")].List...)
	wantExports = append(wantExports, ref(graphs[0]))
	sortRefs(wantExports)
	if !sameRefs(exports, wantExports) || m.Schema != base.Entities[base.Module].Schema || m.Version != base.Entities[base.Module].Version || !bytes.Equal(m.Fields[id("120")].Bytes, base.Entities[base.Module].Fields[id("120")].Bytes) || !sameRefs(m.Fields[id("121")].List, base.Entities[base.Module].Fields[id("121")].List) {
		return fmt.Errorf("package_v3.module")
	}
	items := g.Fields[id("b291")].List
	if !uniqueSorted(items) {
		return fmt.Errorf("package_v3.declarations_order")
	}
	v2member := map[wire.ID]wire.Entity{}
	declarationOwner := map[wire.ID]wire.ID{}
	for _, did := range withSchema(base, id("b021")) {
		for _, member := range base.Entities[did].Fields[id("b211")].List {
			decl := base.Entities[member.Reference].Fields[id("b220")].Reference
			declarationOwner[decl] = did
			v2member[decl] = base.Entities[member.Reference]
		}
	}
	owned := map[wire.ID]wire.ID{}
	used := map[wire.ID]bool{graphs[0]: true}
	details := schemaSet(base, id("b021"))
	origins := schemaSet(e, id("b026"))
	bindings := schemaSet(base, id("b024"))
	for _, v := range items {
		if v.Tag != 6 {
			return fmt.Errorf("package_v3.declaration_ref")
		}
		q, ok := e.Entities[v.Reference]
		if !ok || q.Schema != id("b028") || q.Version != 1 || !declarationShape(q) {
			return fmt.Errorf("package_v3.declaration_schema")
		}
		used[v.Reference] = true
		decl := q.Fields[id("b280")].Reference
		kindWant, rel := relevant[e.Entities[decl].Schema]
		if !rel || owned[decl] != (wire.ID{}) {
			return fmt.Errorf("package_v3.declaration_coverage")
		}
		if prior := declarationOwner[decl]; prior != (wire.ID{}) && prior != q.Fields[id("b281")].Reference {
			return fmt.Errorf("package_v3.annotated_owner")
		}
		if !details[q.Fields[id("b281")].Reference] || !origins[q.Fields[id("b286")].Reference] {
			return fmt.Errorf("package_v3.ownership")
		}
		oid := q.Fields[id("b286")].Reference
		origin := e.Entities[oid]
		if !shape(origin, []wire.ID{id("b260"), id("b261"), id("b262"), id("b263"), id("b264"), id("b265"), id("b266"), id("b267"), id("b268")}, []byte{6, 5, 5, 3, 3, 3, 3, 3, 3}) || len(origin.Fields[id("b262")].Bytes) != sha256.Size || origin.Fields[id("b264")].Unsigned <= origin.Fields[id("b263")].Unsigned || origin.Fields[id("b265")].Unsigned == 0 || origin.Fields[id("b266")].Unsigned == 0 || origin.Fields[id("b267")].Unsigned == 0 || origin.Fields[id("b268")].Unsigned == 0 || e.Entities[origin.Fields[id("b260")].Reference].ID == (wire.ID{}) {
			return fmt.Errorf("package_v3.origin_shape")
		}
		used[oid] = true
		kid := q.Fields[id("b282")].Reference
		k := e.Entities[kid]
		if k.Schema != id("b027") || k.Version != 1 || len(k.Fields) != 1 || k.Fields[id("b270")].Tag != 3 || Kind(k.Fields[id("b270")].Unsigned) != kindWant {
			return fmt.Errorf("package_v3.kind_schema")
		}
		used[kid] = true
		name := string(q.Fields[id("b283")].Bytes)
		if name == "" || !utf8.ValidString(name) || name != semanticName(e.Entities[decl]) {
			return fmt.Errorf("package_v3.name")
		}
		vid := q.Fields[id("b284")].Reference
		vis := e.Entities[vid]
		if vis.Schema != id("b023") || vis.Fields[id("b230")].Tag != 3 || vis.Fields[id("b230")].Unsigned > 2 {
			return fmt.Errorf("package_v3.visibility:%s:%s:%d:%d", vid, vis.Schema, vis.Fields[id("b230")].Tag, vis.Fields[id("b230")].Unsigned)
		}
		used[vid] = true
		_, exported := q.Fields[id("b285")]
		if exported != (vis.Fields[id("b230")].Unsigned > 0) {
			return fmt.Errorf("package_v3.export_visibility")
		}
		if member, ok := v2member[decl]; ok {
			if member.Fields[id("b222")].Reference != vid || member.Fields[id("b224")].Reference != oid {
				return fmt.Errorf("package_v3.annotation_mismatch")
			}
			mv, mexported := member.Fields[id("b223")]
			dv, dexported := q.Fields[id("b285")]
			// Package v2 ownership names receiver callables and concrete
			// realizations uniquely; Package v3 supplies their canonical semantic
			// names. Data/interface declarations retain identical source names.
			if (kindWant == DataType || kindWant == BehavioralInterface) && string(member.Fields[id("b221")].Bytes) != name {
				return fmt.Errorf("package_v3.annotation_mismatch")
			}
			if mexported != dexported || mexported && (kindWant == DataType || kindWant == BehavioralInterface) && !bytes.Equal(mv.Bytes, dv.Bytes) {
				return fmt.Errorf("package_v3.annotation_export")
			}
		}
		for _, im := range q.Fields[id("b287")].List {
			if im.Tag != 6 || !bindings[im.Reference] {
				return fmt.Errorf("package_v3.import")
			}
		}
		if _, has := q.Fields[id("b288")]; has {
			return fmt.Errorf("package_v3.generic_definition_unavailable")
		}
		owned[decl] = q.Fields[id("b281")].Reference
		declarationOwner[decl] = q.Fields[id("b281")].Reference
	}
	for x, q := range e.Entities {
		if _, ok := relevant[q.Schema]; ok && owned[x] == (wire.ID{}) {
			return fmt.Errorf("package_v3.unowned:%s", x)
		}
	}
	for decl, owner := range owned {
		q := e.Entities[decl]
		if q.Schema != id("a002") {
			continue
		}
		receiver := e.Entities[q.Fields[id("a0021")].Reference]
		if declarationOwner[receiver.Fields[id("a0001")].Reference] != owner {
			return fmt.Errorf("package_v3.receiver_owner:%s", decl)
		}
	}
	detailPackage := map[wire.ID]wire.ID{}
	bindingsByEdge := map[wire.ID]map[wire.ID]wire.ID{}
	for did := range details {
		d := base.Entities[did]
		detailPackage[did] = d.Fields[id("b210")].Reference
		edges := map[wire.ID]wire.ID{}
		for _, v := range d.Fields[id("b212")].List {
			b := base.Entities[v.Reference]
			if target, ok := b.Fields[id("b243")]; ok {
				edges[target.Reference] = v.Reference
			}
		}
		bindingsByEdge[did] = edges
	}
	declarations := map[wire.ID]bool{}
	for x := range declarationOwner {
		declarations[x] = true
	}
	for _, v := range items {
		q := e.Entities[v.Reference]
		decl, owner := q.Fields[id("b280")].Reference, q.Fields[id("b281")].Reference
		targets := map[wire.ID]bool{}
		// Execution v35 represents concrete generic arguments directly but has
		// no generic-definition entity. Those references are instantiation
		// parameters, not imports made by the declaring family package.
		if relevant[e.Entities[decl].Schema] != GenericRealization {
			targets = referencedOwners(e, decl, declarations, declarationOwner, owner)
		}
		want := []wire.ID{}
		for target := range targets {
			binding, ok := bindingsByEdge[owner][detailPackage[target]]
			if !ok {
				return fmt.Errorf("package_v3.missing_import:%s", decl)
			}
			want = append(want, binding)
		}
		sort.Slice(want, func(i, j int) bool { return bytes.Compare(want[i][:], want[j][:]) < 0 })
		got := q.Fields[id("b287")].List
		if len(got) != len(want) {
			return fmt.Errorf("package_v3.import_exactness:%s", decl)
		}
		for i := range want {
			if got[i].Reference != want[i] {
				return fmt.Errorf("package_v3.import_exactness:%s", decl)
			}
		}
	}
	for _, s := range []wire.ID{id("b027"), id("b028"), id("b029")} {
		for _, x := range withSchema(e, s) {
			if !used[x] {
				return fmt.Errorf("package_v3.orphan:%s", x)
			}
		}
	}
	if !bytes.Equal(g.Fields[id("b292")].Bytes, snapshotRevision(e, graphs[0])) {
		return fmt.Errorf("package_v3.revision")
	}
	return exactBase(base, e, used)
}

func validatedBase(in Inputs) (wire.Envelope, error) {
	if !in.Contracts.Validated() {
		return wire.Envelope{}, fmt.Errorf("package_v3.contracts")
	}
	if err := packagedetailinstance.Validate(in.PackageV2); err != nil {
		return wire.Envelope{}, err
	}
	return wire.Decode(in.PackageV2)
}
func upgrade(e *wire.Envelope) error {
	b003 := id("b003")
	for x, q := range e.Entities {
		if q.Schema == id("13") && q.Fields[id("130")].Reference == id("b000") {
			q.Fields[id("131")] = blob(b003[:])
			e.Entities[x] = q
			return nil
		}
	}
	return fmt.Errorf("package_v3.package_import")
}
func checkUpgrade(e wire.Envelope) error {
	b003 := id("b003")
	for _, q := range e.Entities {
		if q.Schema == id("13") && q.Fields[id("130")].Reference == id("b000") {
			if bytes.Equal(q.Fields[id("131")].Bytes, b003[:]) {
				return nil
			}
		}
	}
	return fmt.Errorf("package_v3.package_pin")
}
func snapshotRevision(e wire.Envelope, root wire.ID) []byte {
	seen := map[wire.ID]bool{}
	todo := []wire.ID{root}
	closure := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		q, ok := e.Entities[x]
		if !ok {
			return nil
		}
		fields := map[wire.ID]wire.Value{}
		for f, v := range q.Fields {
			if x == root && f == id("b292") {
				continue
			}
			fields[f] = v
			collect(v, &todo)
		}
		q.Fields = fields
		closure.Entities[x] = q
	}
	b, _ := wire.Encode(closure)
	h := sha256.Sum256(append([]byte("seme.complete-package-graph.v1\x00"), b...))
	return h[:]
}
func artifactRevision(e wire.Envelope) wire.ID {
	e.Revision = wire.ID{}
	b, _ := wire.Encode(e)
	h := sha256.Sum256(append([]byte("seme.package-v3.artifact.v1\x00"), b...))
	var x wire.ID
	copy(x[:], h[:16])
	return x
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
func referencedOwners(e wire.Envelope, root wire.ID, declarations map[wire.ID]bool, owners map[wire.ID]wire.ID, owner wire.ID) map[wire.ID]bool {
	out := map[wire.ID]bool{}
	seen := map[wire.ID]bool{}
	todo := []wire.ID{root}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if seen[x] {
			continue
		}
		seen[x] = true
		if x != root && declarations[x] {
			if owners[x] != owner {
				out[owners[x]] = true
			}
			continue
		}
		q, ok := e.Entities[x]
		if !ok {
			continue
		}
		for _, v := range q.Fields {
			collect(v, &todo)
		}
	}
	return out
}
func stable(raw []byte, parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.package-v3-instance.identity.v1\x00"))
	sum := sha256.Sum256(raw)
	h.Write(sum[:])
	for _, p := range parts {
		_ = binary.Write(h, binary.BigEndian, uint64(len(p)))
		h.Write([]byte(p))
	}
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}
func visibility(e wire.Envelope, seed []byte, code uint64) (wire.ID, error) {
	for x, q := range e.Entities {
		if q.Schema == id("b023") && q.Fields[id("b230")].Unsigned == code {
			return x, nil
		}
	}
	x := stable(seed, "visibility", fmt.Sprint(code))
	if _, exists := e.Entities[x]; exists {
		return wire.ID{}, fmt.Errorf("package_v3.identity_collision")
	}
	e.Entities[x] = wire.Entity{ID: x, Schema: id("b023"), Version: 1, Fields: map[wire.ID]wire.Value{id("b230"): u(code)}}
	return x, nil
}
func semanticName(q wire.Entity) string {
	for _, f := range []wire.ID{id("9300"), id("a0100"), id("a0020")} {
		if v, ok := q.Fields[f]; ok && v.Tag == 5 {
			return string(v.Bytes)
		}
	}
	if relevant[q.Schema] == GenericRealization {
		return q.ID.String()
	}
	return ""
}
func declarationShape(q wire.Entity) bool {
	required := map[wire.ID]byte{id("b280"): 6, id("b281"): 6, id("b282"): 6, id("b283"): 5, id("b284"): 6, id("b286"): 6, id("b287"): 7}
	if len(q.Fields) < len(required) || len(q.Fields) > len(required)+2 {
		return false
	}
	for f, t := range required {
		if q.Fields[f].Tag != t {
			return false
		}
	}
	for f, v := range q.Fields {
		if _, ok := required[f]; ok {
			continue
		}
		if (f != id("b285") || v.Tag != 5) && (f != id("b288") || v.Tag != 6) {
			return false
		}
	}
	return true
}
func exactBase(base, out wire.Envelope, added map[wire.ID]bool) error {
	for x, q := range base.Entities {
		oq, ok := out.Entities[x]
		if !ok {
			return fmt.Errorf("package_v3.base_missing")
		}
		if x == base.Module {
			continue
		}
		if q.Schema == id("13") && q.Fields[id("130")].Reference == id("b000") {
			got := out.Entities[x]
			if got.Schema != q.Schema || got.Version != q.Version || got.Fields[id("130")].Reference != q.Fields[id("130")].Reference || len(got.Fields) != len(q.Fields) {
				return fmt.Errorf("package_v3.package_import_changed")
			}
			continue
		}
		a, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{x: q}})
		b, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{x: oq}})
		if !bytes.Equal(a, b) {
			return fmt.Errorf("package_v3.base_changed:%s", x)
		}
	}
	for x, q := range out.Entities {
		if _, ok := base.Entities[x]; ok {
			continue
		}
		if !added[x] {
			return fmt.Errorf("package_v3.extra:%s:%s", x, q.Schema)
		}
	}
	return nil
}
func cloneEnvelope(e wire.Envelope) wire.Envelope {
	b, _ := wire.Encode(e)
	x, _ := wire.Decode(b)
	return x
}
func schemaSet(e wire.Envelope, s wire.ID) map[wire.ID]bool {
	m := map[wire.ID]bool{}
	for x, q := range e.Entities {
		if q.Schema == s {
			m[x] = true
		}
	}
	return m
}
func withSchema(e wire.Envelope, s wire.ID) []wire.ID {
	var xs []wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			xs = append(xs, x)
		}
	}
	return xs
}
func one(e wire.Envelope, s wire.ID) wire.ID {
	xs := withSchema(e, s)
	if len(xs) == 1 {
		return xs[0]
	}
	return wire.ID{}
}
func shape(q wire.Entity, fs []wire.ID, tags []byte) bool {
	if len(q.Fields) != len(fs) {
		return false
	}
	for i, f := range fs {
		if q.Fields[f].Tag != tags[i] {
			return false
		}
	}
	return true
}
func ref(x wire.ID) wire.Value { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func u(x uint64) wire.Value    { return wire.Value{Tag: 3, Unsigned: x} }
func sortRefs(v []wire.Value) {
	sort.Slice(v, func(i, j int) bool { return bytes.Compare(v[i].Reference[:], v[j].Reference[:]) < 0 })
}
func unique(v []wire.Value) bool {
	m := map[wire.ID]bool{}
	for _, x := range v {
		if m[x.Reference] {
			return false
		}
		m[x.Reference] = true
	}
	return true
}
func uniqueSorted(v []wire.Value) bool {
	for i, x := range v {
		if x.Tag != 6 || (i > 0 && bytes.Compare(v[i-1].Reference[:], x.Reference[:]) >= 0) {
			return false
		}
	}
	return true
}
func sameRefs(a, b []wire.Value) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Tag != 6 || b[i].Tag != 6 || a[i].Reference != b[i].Reference {
			return false
		}
	}
	return true
}
func sameEntity(a, b wire.Entity) bool {
	x, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{a.ID: a}})
	y, _ := wire.Encode(wire.Envelope{Entities: map[wire.ID]wire.Entity{b.ID: b}})
	return bytes.Equal(x, y)
}
func sameOrigin(a, b wire.Entity) bool { a.ID = b.ID; return sameEntity(a, b) }
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
