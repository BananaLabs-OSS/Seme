// Package packagedetailinstance validates Package Contract v2 instance graphs.
package packagedetailinstance

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path"
	"sort"
	"strings"

	"seme.local/reference/packageinstance"
	"seme.local/reference/wire"
)

var (
	moduleSchema     = id("12")
	importSchema     = id("13")
	packageSchema    = id("b010")
	interfaceSchema  = id("b011")
	dependencySchema = id("b012")
	graphSchema      = id("b020")
	detailSchema     = id("b021")
	memberSchema     = id("b022")
	visibilitySchema = id("b023")
	bindingSchema    = id("b024")
	classSchema      = id("b025")
	originSchema     = id("b026")
)

func Validate(source []byte) error {
	return validate(source, false)
}

func ValidateV4(source []byte) error { return validate(source, true) }

func validate(source []byte, v4 bool) error {
	var baseErr error
	if v4 {
		baseErr = packageinstance.ValidateV4(source)
	} else {
		baseErr = packageinstance.Validate(source)
	}
	if baseErr != nil {
		return fmt.Errorf("package_detail.v1:%w", baseErr)
	}
	e, err := wire.Decode(source)
	if err != nil {
		return err
	}
	canonical, err := wire.Encode(e)
	if err != nil || !bytes.Equal(canonical, source) {
		return fmt.Errorf("package_detail.noncanonical")
	}
	revision := id("b002")
	if v4 {
		revision = id("b004")
	}
	if !hasPin(e, revision) {
		return fmt.Errorf("package_detail.v2_import_unpinned")
	}
	graphs := idsWithSchema(e, graphSchema)
	if len(graphs) != 1 {
		return fmt.Errorf("package_detail.graph_count:%d", len(graphs))
	}
	graph := e.Entities[graphs[0]]
	if err = shape(graph, []wire.ID{id("b200"), id("b201")}, []byte{7, 5}); err != nil {
		return at(graph.ID, err)
	}
	details, err := refs(graph, id("b200"))
	if err != nil || len(details) == 0 || !sortedUnique(details) {
		return fmt.Errorf("package_detail.details_unsorted")
	}
	packages := idsWithSchema(e, packageSchema)
	if len(details) != len(packages) {
		return fmt.Errorf("package_detail.package_coverage")
	}
	packageSet := set(packages)
	detailPackages := map[wire.ID]bool{}
	declarationOwners := map[wire.ID]wire.ID{}
	originOwners := map[wire.ID]wire.ID{}
	programs := idsWithSchema(e, id("9015"))
	if len(programs) != 1 {
		return fmt.Errorf("package_detail.program_count:%d", len(programs))
	}
	programFunctions, err := refs(e.Entities[programs[0]], id("9150"))
	if err != nil || len(programFunctions) == 0 {
		return fmt.Errorf("package_detail.program_members")
	}
	programSet := set(programFunctions)
	used := map[wire.ID]bool{graph.ID: true}
	for _, did := range details {
		d, ok := e.Entities[did]
		if !ok || d.Schema != detailSchema {
			return fmt.Errorf("package_detail.detail_schema:%s", did)
		}
		used[did] = true
		if err = shape(d, []wire.ID{id("b210"), id("b211"), id("b212"), id("b213"), id("b214")}, []byte{6, 7, 7, 7, 5}); err != nil {
			return at(did, err)
		}
		pid, _ := ref(d, id("b210"))
		if !packageSet[pid] || detailPackages[pid] {
			return fmt.Errorf("package_detail.package_ownership:%s", pid)
		}
		detailPackages[pid] = true
		members, _ := refs(d, id("b211"))
		bindings, _ := refs(d, id("b212"))
		origins, _ := refs(d, id("b213"))
		if !sortedUnique(members) || !sortedUnique(bindings) || !sortedUnique(origins) {
			return fmt.Errorf("package_detail.refs_unsorted:%s", did)
		}
		originSet := set(origins)
		for _, oid := range origins {
			if prior, exists := originOwners[oid]; exists {
				return fmt.Errorf("package_detail.origin_owned_twice:%s:%s", prior, oid)
			}
			originOwners[oid] = did
			if err = validateOrigin(e, oid); err != nil {
				return err
			}
			used[oid] = true
		}
		exports := map[string]wire.ID{}
		exportNames := map[string]bool{}
		names := map[string]bool{}
		for _, mid := range members {
			m, ok := e.Entities[mid]
			if !ok || m.Schema != memberSchema {
				return fmt.Errorf("package_detail.member_schema:%s", mid)
			}
			used[mid] = true
			if err = shapeOptional(m, []wire.ID{id("b220"), id("b221"), id("b222"), id("b224")}, id("b223")); err != nil {
				return at(mid, err)
			}
			decl, _ := ref(m, id("b220"))
			if _, ok = e.Entities[decl]; !ok {
				return fmt.Errorf("package_detail.declaration_missing:%s", decl)
			}
			if prior, ok := declarationOwners[decl]; ok {
				return fmt.Errorf("package_detail.declaration_owned_twice:%s:%s", prior, decl)
			}
			declarationOwners[decl] = pid
			if e.Entities[decl].Schema == id("9011") && !programSet[decl] {
				return fmt.Errorf("package_detail.function_outside_program:%s", decl)
			}
			name, _ := blob(m, id("b221"))
			if len(name) == 0 || names[string(name)] {
				return fmt.Errorf("package_detail.member_name:%s", mid)
			}
			names[string(name)] = true
			vid, _ := ref(m, id("b222"))
			code, er := validateEnum(e, vid, visibilitySchema, id("b230"), 2)
			if er != nil {
				return er
			}
			used[vid] = true
			oid, _ := ref(m, id("b224"))
			if !originSet[oid] {
				return fmt.Errorf("package_detail.member_origin:%s", mid)
			}
			value, exported := m.Fields[id("b223")]
			if code == 0 && exported || code > 0 && !exported {
				return fmt.Errorf("package_detail.export_visibility:%s", mid)
			}
			if exported {
				if value.Tag != 5 || len(value.Bytes) == 0 {
					return fmt.Errorf("package_detail.export_visibility:%s", mid)
				}
				if exportNames[string(value.Bytes)] {
					return fmt.Errorf("package_detail.export_duplicate:%s", value.Bytes)
				}
				exportNames[string(value.Bytes)] = true
				// Package v1 TypedInterface represents callable exports only.
				// Package v2 may additionally own visible data types and receiver
				// methods; their exact visibility is annotated by Package v3.
				if e.Entities[decl].Schema == id("9011") {
					exports[string(value.Bytes)] = decl
				}
			}
		}
		if err = matchInterfaces(e, pid, exports); err != nil {
			return err
		}
		deps, er := packageDependencies(e, pid)
		if er != nil {
			return er
		}
		aliases := map[string]bool{}
		for _, bid := range bindings {
			b, ok := e.Entities[bid]
			if !ok || b.Schema != bindingSchema {
				return fmt.Errorf("package_detail.binding_schema:%s", bid)
			}
			used[bid] = true
			if err = shapeBinding(b); err != nil {
				return at(bid, err)
			}
			alias := optionalBlob(b, id("b240"))
			if alias != "" && aliases[alias] {
				return fmt.Errorf("package_detail.alias_duplicate:%s", alias)
			}
			aliases[alias] = alias != ""
			requested, _ := blob(b, id("b241"))
			if len(requested) == 0 {
				return fmt.Errorf("package_detail.requested_empty:%s", bid)
			}
			cid, _ := ref(b, id("b242"))
			class, er := validateEnum(e, cid, classSchema, id("b250"), 1)
			if er != nil {
				return er
			}
			used[cid] = true
			oid, _ := ref(b, id("b245"))
			if !originSet[oid] {
				return fmt.Errorf("package_detail.binding_origin:%s", bid)
			}
			local, lok := optionalRef(b, id("b243"))
			external, eok := optionalRef(b, id("b244"))
			if class == 0 {
				if !lok || eok || !packageSet[local] || local == pid {
					return fmt.Errorf("package_detail.local_arm:%s", bid)
				}
				dep, ok := deps[local]
				if !ok || string(requested) != dep.name || dep.requirement != "local" {
					return fmt.Errorf("package_detail.local_dependency:%s", bid)
				}
			} else {
				if lok || !eok || e.Entities[external].Schema != dependencySchema {
					return fmt.Errorf("package_detail.external_arm:%s", bid)
				}
				if !ownsDependency(e, pid, external) {
					return fmt.Errorf("package_detail.external_dependency:%s", bid)
				}
				dep := dependency(e, external)
				if string(requested) != dep.name || dep.requirement == "" || dep.requirement == "local" {
					return fmt.Errorf("package_detail.external_identity:%s", bid)
				}
			}
		}
		want, er := DetailRevision(e, did)
		if er != nil {
			return er
		}
		got, _ := blob(d, id("b214"))
		if !bytes.Equal(got, want) {
			return fmt.Errorf("package_detail.detail_revision:%s", did)
		}
	}
	for _, function := range programFunctions {
		if _, owned := declarationOwners[function]; !owned {
			return fmt.Errorf("package_detail.program_function_unowned:%s", function)
		}
	}
	for _, schema := range []wire.ID{detailSchema, memberSchema, visibilitySchema, bindingSchema, classSchema, originSchema} {
		for _, x := range idsWithSchema(e, schema) {
			if !used[x] {
				return fmt.Errorf("package_detail.orphan:%s", x)
			}
		}
	}
	want, err := ContentRevision(e, graph.ID)
	if err != nil {
		return err
	}
	got, _ := blob(graph, id("b201"))
	if !bytes.Equal(got, want) {
		return fmt.Errorf("package_detail.content_revision")
	}
	return nil
}

func DetailRevision(e wire.Envelope, detail wire.ID) ([]byte, error) {
	return revision(e, detail, id("b214"), "seme.package-detail.v2\x00")
}
func ContentRevision(e wire.Envelope, graph wire.ID) ([]byte, error) {
	return revision(e, graph, id("b201"), "seme.package-graph.v2\x00")
}
func revision(e wire.Envelope, root, excluded wire.ID, domain string) ([]byte, error) {
	seen := map[wire.ID]bool{}
	q := []wire.ID{root}
	closure := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	for len(q) > 0 {
		x := q[0]
		q = q[1:]
		if seen[x] {
			continue
		}
		v, ok := e.Entities[x]
		if !ok {
			return nil, fmt.Errorf("package_detail.revision_missing:%s", x)
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
		closure.Entities[x] = v
	}
	raw, err := wire.Encode(closure)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write([]byte(domain))
	h.Write(raw)
	return h.Sum(nil), nil
}

func hasPin(e wire.Envelope, revision wire.ID) bool {
	m := e.Entities[e.Module]
	if m.Schema != moduleSchema {
		return false
	}
	xs, err := refs(m, id("121"))
	if err != nil {
		return false
	}
	for _, x := range xs {
		q := e.Entities[x]
		if q.Schema == importSchema {
			a, _ := ref(q, id("130"))
			b, _ := blob(q, id("131"))
			if a == id("b000") && bytes.Equal(b, revision[:]) {
				return true
			}
		}
	}
	return false
}
func matchInterfaces(e wire.Envelope, pid wire.ID, exports map[string]wire.ID) error {
	p := e.Entities[pid]
	xs, _ := refs(p, id("b102"))
	if len(xs) != len(exports) {
		return fmt.Errorf("package_detail.export_set:%s", pid)
	}
	for _, x := range xs {
		q := e.Entities[x]
		if q.Schema != interfaceSchema {
			return fmt.Errorf("package_detail.interface_schema")
		}
		name, _ := blob(q, id("b110"))
		fn, _ := ref(q, id("b111"))
		if exports[string(name)] != fn {
			return fmt.Errorf("package_detail.export_set:%s", pid)
		}
	}
	return nil
}

type dependencyInfo struct{ name, requirement string }

func dependency(e wire.Envelope, did wire.ID) dependencyInfo {
	q := e.Entities[did]
	name, _ := blob(q, id("b120"))
	requirement, _ := blob(q, id("b121"))
	return dependencyInfo{string(name), string(requirement)}
}
func packageDependencies(e wire.Envelope, pid wire.ID) (map[wire.ID]dependencyInfo, error) {
	out := map[wire.ID]dependencyInfo{}
	names := map[string]bool{}
	xs, _ := refs(e.Entities[pid], id("b103"))
	for _, x := range xs {
		q := e.Entities[x]
		target, _ := ref(q, id("b122"))
		info := dependency(e, x)
		if _, exists := out[target]; exists {
			return nil, fmt.Errorf("package_detail.dependency_target_duplicate:%s", target)
		}
		if names[info.name] {
			return nil, fmt.Errorf("package_detail.dependency_name_duplicate:%s", info.name)
		}
		out[target], names[info.name] = info, true
	}
	return out, nil
}
func ownsDependency(e wire.Envelope, pid, did wire.ID) bool {
	xs, _ := refs(e.Entities[pid], id("b103"))
	for _, x := range xs {
		if x == did {
			return true
		}
	}
	return false
}
func validateOrigin(e wire.Envelope, oid wire.ID) error {
	o, ok := e.Entities[oid]
	if !ok || o.Schema != originSchema {
		return fmt.Errorf("package_detail.origin_schema:%s", oid)
	}
	fs := []wire.ID{id("b260"), id("b261"), id("b262"), id("b263"), id("b264"), id("b265"), id("b266"), id("b267"), id("b268")}
	tags := []byte{6, 5, 5, 3, 3, 3, 3, 3, 3}
	if err := shape(o, fs, tags); err != nil {
		return at(oid, err)
	}
	p, _ := blob(o, id("b261"))
	s := string(p)
	if s == "" || strings.Contains(s, "\\") || path.IsAbs(s) || path.Clean(s) != s || strings.HasPrefix(s, "../") {
		return fmt.Errorf("package_detail.origin_path:%s", oid)
	}
	digest, _ := blob(o, id("b262"))
	if len(digest) != sha256.Size {
		return fmt.Errorf("package_detail.origin_digest:%s", oid)
	}
	start := o.Fields[id("b263")].Unsigned
	end := o.Fields[id("b264")].Unsigned
	sl, sc, el, ec := o.Fields[id("b265")].Unsigned, o.Fields[id("b266")].Unsigned, o.Fields[id("b267")].Unsigned, o.Fields[id("b268")].Unsigned
	if end <= start || sl == 0 || sc == 0 || el == 0 || ec == 0 || el < sl || el == sl && ec <= sc {
		return fmt.Errorf("package_detail.origin_range:%s", oid)
	}
	if _, ok = e.Entities[o.Fields[id("b260")].Reference]; !ok {
		return fmt.Errorf("package_detail.source_identity:%s", oid)
	}
	return nil
}
func validateEnum(e wire.Envelope, x, schema, field wire.ID, max uint64) (uint64, error) {
	q, ok := e.Entities[x]
	if !ok || q.Schema != schema {
		return 0, fmt.Errorf("package_detail.enum_schema:%s", x)
	}
	if err := shape(q, []wire.ID{field}, []byte{3}); err != nil {
		return 0, at(x, err)
	}
	v := q.Fields[field].Unsigned
	if v > max {
		return 0, fmt.Errorf("package_detail.enum:%s", x)
	}
	return v, nil
}
func shape(x wire.Entity, fs []wire.ID, tags []byte) error {
	if x.Version != 1 || len(x.Fields) != len(fs) {
		return fmt.Errorf("shape")
	}
	for i, f := range fs {
		v, ok := x.Fields[f]
		if !ok || v.Tag != tags[i] {
			return fmt.Errorf("field:%s", f)
		}
	}
	return nil
}
func shapeOptional(x wire.Entity, required []wire.ID, optional wire.ID) error {
	if x.Version != 1 || len(x.Fields) < len(required) || len(x.Fields) > len(required)+1 {
		return fmt.Errorf("shape")
	}
	for _, f := range required {
		if _, ok := x.Fields[f]; !ok {
			return fmt.Errorf("field:%s", f)
		}
	}
	if x.Fields[required[0]].Tag != 6 || x.Fields[required[1]].Tag != 5 || x.Fields[required[2]].Tag != 6 || x.Fields[required[3]].Tag != 6 {
		return fmt.Errorf("kind")
	}
	if v, ok := x.Fields[optional]; ok && v.Tag != 5 {
		return fmt.Errorf("kind")
	}
	return nil
}
func shapeBinding(x wire.Entity) error {
	required := []wire.ID{id("b241"), id("b242"), id("b245")}
	allowed := map[wire.ID]byte{id("b240"): 5, id("b241"): 5, id("b242"): 6, id("b243"): 6, id("b244"): 6, id("b245"): 6}
	if x.Version != 1 || len(x.Fields) < 3 || len(x.Fields) > 6 {
		return fmt.Errorf("shape")
	}
	for _, f := range required {
		if _, ok := x.Fields[f]; !ok {
			return fmt.Errorf("field:%s", f)
		}
	}
	for f, v := range x.Fields {
		tag, ok := allowed[f]
		if !ok || tag != v.Tag {
			return fmt.Errorf("field:%s", f)
		}
	}
	return nil
}
func idsWithSchema(e wire.Envelope, s wire.ID) []wire.ID {
	var out []wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i][:], out[j][:]) < 0 })
	return out
}
func refs(x wire.Entity, f wire.ID) ([]wire.ID, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 7 {
		return nil, fmt.Errorf("kind")
	}
	out := make([]wire.ID, len(v.List))
	for i, z := range v.List {
		if z.Tag != 6 {
			return nil, fmt.Errorf("list_kind")
		}
		out[i] = z.Reference
	}
	return out, nil
}
func ref(x wire.Entity, f wire.ID) (wire.ID, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 6 {
		return wire.ID{}, fmt.Errorf("kind")
	}
	return v.Reference, nil
}
func blob(x wire.Entity, f wire.ID) ([]byte, error) {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 5 {
		return nil, fmt.Errorf("kind")
	}
	return v.Bytes, nil
}
func optionalRef(x wire.Entity, f wire.ID) (wire.ID, bool) {
	v, ok := x.Fields[f]
	return v.Reference, ok && v.Tag == 6
}
func optionalBlob(x wire.Entity, f wire.ID) string {
	v, ok := x.Fields[f]
	if !ok || v.Tag != 5 {
		return ""
	}
	return string(v.Bytes)
}
func sortedUnique(xs []wire.ID) bool {
	for i := 1; i < len(xs); i++ {
		if bytes.Compare(xs[i-1][:], xs[i][:]) >= 0 {
			return false
		}
	}
	return true
}
func set(xs []wire.ID) map[wire.ID]bool {
	m := map[wire.ID]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
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
func at(x wire.ID, err error) error { return fmt.Errorf("package_detail.field:%s:%w", x, err) }
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
