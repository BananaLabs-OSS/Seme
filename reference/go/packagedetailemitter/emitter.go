package packagedetailemitter

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/wire"
	"sort"
)

var packageSchema = id("b010")

// Emit preserves base and adds one deterministic, independently validated v2 graph.
func Emit(base wire.Envelope, g packagedetail.Graph) ([]byte, error) {
	return emit(base, g, false)
}

// EmitV4 adds Package detail semantics to a fresh b004-authorized v36 base.
func EmitV4(base wire.Envelope, g packagedetail.Graph) ([]byte, error) {
	return emit(base, g, true)
}

func emit(base wire.Envelope, g packagedetail.Graph, v4 bool) ([]byte, error) {
	if err := packagedetail.Validate(g); err != nil {
		return nil, err
	}
	e, err := clone(base)
	if err != nil {
		return nil, fmt.Errorf("package_detail_emitter.base:%w", err)
	}
	if err = upgradePackagePin(&e, v4); err != nil {
		return nil, err
	}
	packages := map[string]wire.ID{}
	deps := map[string]map[wire.ID]bool{}
	for x, q := range e.Entities {
		if q.Schema == packageSchema {
			n := string(q.Fields[id("b100")].Bytes)
			if n == "" || packages[n] != (wire.ID{}) {
				return nil, fmt.Errorf("package_detail_emitter.package_identity")
			}
			packages[n] = x
			deps[n] = map[wire.ID]bool{}
			for _, v := range q.Fields[id("b103")].List {
				deps[n][v.Reference] = true
			}
		}
	}
	if len(g.Packages) != len(packages) {
		return nil, fmt.Errorf("package_detail_emitter.package_coverage")
	}
	seen := map[string]bool{}
	detailRefs := []wire.Value{}
	vis := map[packagedetail.Visibility]wire.ID{}
	classes := map[packagedetail.ImportClass]wire.ID{}
	for _, p := range g.Packages {
		pid, ok := packages[p.Identity]
		if !ok || seen[p.Identity] {
			return nil, fmt.Errorf("package_detail_emitter.package:%s", p.Identity)
		}
		seen[p.Identity] = true
		originIDs := map[string]wire.ID{}
		origins, members, imports := []wire.Value{}, []wire.Value{}, []wire.Value{}
		emitOrigin := func(o packagedetail.Origin) (wire.ID, error) {
			key := originKey(o)
			if x, ok := originIDs[key]; ok {
				return x, nil
			}
			source, err := wire.ParseID(o.SourceIdentity)
			if err != nil {
				return wire.ID{}, fmt.Errorf("package_detail_emitter.source_identity:%w", err)
			}
			x := stable("origin", p.Identity, key)
			q := wire.Entity{ID: x, Schema: id("b026"), Version: 1, Fields: map[wire.ID]wire.Value{id("b260"): ref(source), id("b261"): blob([]byte(o.Path)), id("b262"): blob(o.ContentDigest[:]), id("b263"): u(o.ByteStart), id("b264"): u(o.ByteEnd), id("b265"): u(uint64(o.StartLine)), id("b266"): u(uint64(o.StartColumn)), id("b267"): u(uint64(o.EndLine)), id("b268"): u(uint64(o.EndColumn))}}
			if err = put(e.Entities, q); err != nil {
				return wire.ID{}, err
			}
			originIDs[key] = x
			origins = append(origins, ref(x))
			return x, nil
		}
		for _, m := range p.Members {
			oid, err := emitOrigin(m.Origin)
			if err != nil {
				return nil, err
			}
			decl, err := wire.ParseID(m.Identity)
			if err != nil {
				return nil, fmt.Errorf("package_detail_emitter.member_identity:%w", err)
			}
			vid, ok := vis[m.Visibility]
			if !ok {
				vid = stable("visibility", fmt.Sprint(m.Visibility))
				if err = put(e.Entities, wire.Entity{ID: vid, Schema: id("b023"), Version: 1, Fields: map[wire.ID]wire.Value{id("b230"): u(uint64(m.Visibility))}}); err != nil {
					return nil, err
				}
				vis[m.Visibility] = vid
			}
			mid := stable("member", p.Identity, m.Identity)
			fs := map[wire.ID]wire.Value{id("b220"): ref(decl), id("b221"): blob([]byte(m.Name)), id("b222"): ref(vid), id("b224"): ref(oid)}
			if m.ExportName != "" {
				fs[id("b223")] = blob([]byte(m.ExportName))
			}
			if err = put(e.Entities, wire.Entity{ID: mid, Schema: id("b022"), Version: 1, Fields: fs}); err != nil {
				return nil, err
			}
			members = append(members, ref(mid))
		}
		for _, im := range p.Imports {
			oid, err := emitOrigin(im.Origin)
			if err != nil {
				return nil, err
			}
			cid, ok := classes[im.Class]
			if !ok {
				cid = stable("class", fmt.Sprint(im.Class))
				if err = put(e.Entities, wire.Entity{ID: cid, Schema: id("b025"), Version: 1, Fields: map[wire.ID]wire.Value{id("b250"): u(uint64(im.Class))}}); err != nil {
					return nil, err
				}
				classes[im.Class] = cid
			}
			iid := stable("import", p.Identity, im.Origin.SourceIdentity, fmt.Sprint(im.Origin.ByteStart), im.Requested, im.Alias)
			fs := map[wire.ID]wire.Value{id("b241"): blob([]byte(im.Requested)), id("b242"): ref(cid), id("b245"): ref(oid)}
			if im.Alias != "" {
				fs[id("b240")] = blob([]byte(im.Alias))
			}
			if im.Class == packagedetail.Local {
				target, yes := packages[im.Resolved]
				if !yes {
					return nil, fmt.Errorf("package_detail_emitter.local_package:%s", im.Resolved)
				}
				fs[id("b243")] = ref(target)
			} else {
				dep, er := wire.ParseID(im.Resolved)
				if er != nil || !deps[p.Identity][dep] {
					return nil, fmt.Errorf("package_detail_emitter.external_dependency:%s", im.Resolved)
				}
				fs[id("b244")] = ref(dep)
			}
			if err = put(e.Entities, wire.Entity{ID: iid, Schema: id("b024"), Version: 1, Fields: fs}); err != nil {
				return nil, err
			}
			imports = append(imports, ref(iid))
		}
		sortRefs(origins)
		sortRefs(members)
		sortRefs(imports)
		did := stable("detail", p.Identity)
		d := wire.Entity{ID: did, Schema: id("b021"), Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): ref(pid), id("b211"): list(members), id("b212"): list(imports), id("b213"): list(origins), id("b214"): blob(make([]byte, 32))}}
		if err := put(e.Entities, d); err != nil {
			return nil, err
		}
		r, err := packagedetailinstance.DetailRevision(e, did)
		if err != nil {
			return nil, err
		}
		d.Fields[id("b214")] = blob(r)
		e.Entities[did] = d
		detailRefs = append(detailRefs, ref(did))
	}
	sortRefs(detailRefs)
	gid := stable("graph")
	graph := wire.Entity{ID: gid, Schema: id("b020"), Version: 1, Fields: map[wire.ID]wire.Value{id("b200"): list(detailRefs), id("b201"): blob(make([]byte, 32))}}
	if err := put(e.Entities, graph); err != nil {
		return nil, err
	}
	r, err := packagedetailinstance.ContentRevision(e, gid)
	if err != nil {
		return nil, err
	}
	graph.Fields[id("b201")] = blob(r)
	e.Entities[gid] = graph
	out, err := wire.Encode(e)
	if err != nil {
		return nil, err
	}
	if v4 {
		err = packagedetailinstance.ValidateV4(out)
	} else {
		err = packagedetailinstance.Validate(out)
	}
	if err != nil {
		return nil, fmt.Errorf("package_detail_emitter.validate:%w", err)
	}
	return out, nil
}

func upgradePackagePin(e *wire.Envelope, v4 bool) error {
	module, ok := e.Entities[e.Module]
	if !ok || module.Schema != id("12") {
		return fmt.Errorf("package_detail_emitter.module")
	}
	imports, ok := module.Fields[id("121")]
	if !ok || imports.Tag != 7 {
		return fmt.Errorf("package_detail_emitter.imports")
	}
	var packageImports []wire.ID
	for _, value := range imports.List {
		if value.Tag != 6 {
			return fmt.Errorf("package_detail_emitter.import_reference")
		}
		item, exists := e.Entities[value.Reference]
		if !exists || item.Schema != id("13") {
			continue
		}
		moduleValue, mok := item.Fields[id("130")]
		if mok && moduleValue.Tag == 6 && moduleValue.Reference == id("b000") {
			packageImports = append(packageImports, item.ID)
		}
	}
	if len(packageImports) != 1 {
		return fmt.Errorf("package_detail_emitter.package_pin_count:%d", len(packageImports))
	}
	item := e.Entities[packageImports[0]]
	revision, ok := item.Fields[id("131")]
	b001, b002 := id("b001"), id("b002")
	if v4 {
		b004 := id("b004")
		if !ok || revision.Tag != 5 || !bytes.Equal(revision.Bytes, b004[:]) {
			return fmt.Errorf("package_detail_emitter.package_pin_revision")
		}
		return nil
	}
	if !ok || revision.Tag != 5 || len(revision.Bytes) != len(wire.ID{}) || !bytes.Equal(revision.Bytes, b001[:]) && !bytes.Equal(revision.Bytes, b002[:]) {
		return fmt.Errorf("package_detail_emitter.package_pin_revision")
	}
	item.Fields[id("131")] = blob(b002[:])
	e.Entities[item.ID] = item
	return nil
}

func stable(parts ...string) wire.ID {
	h := sha256.New()
	h.Write([]byte("seme.package-detail-emitter.v1\x00"))
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	var x wire.ID
	copy(x[:], h.Sum(nil))
	return x
}
func originKey(o packagedetail.Origin) string {
	return fmt.Sprintf("%s\x00%s\x00%x\x00%d:%d:%d:%d:%d:%d", o.SourceIdentity, o.Path, o.ContentDigest, o.ByteStart, o.ByteEnd, o.StartLine, o.StartColumn, o.EndLine, o.EndColumn)
}
func put(m map[wire.ID]wire.Entity, q wire.Entity) error {
	if _, ok := m[q.ID]; ok {
		return fmt.Errorf("package_detail_emitter.identity_collision:%s", q.ID)
	}
	m[q.ID] = q
	return nil
}
func clone(e wire.Envelope) (wire.Envelope, error) {
	raw, err := wire.Encode(e)
	if err != nil {
		return wire.Envelope{}, err
	}
	out, err := wire.Decode(raw)
	if err != nil {
		return wire.Envelope{}, err
	}
	return out, nil
}
func ref(x wire.ID) wire.Value       { return wire.Value{Tag: 6, Reference: x} }
func blob(x []byte) wire.Value       { return wire.Value{Tag: 5, Bytes: append([]byte(nil), x...)} }
func u(x uint64) wire.Value          { return wire.Value{Tag: 3, Unsigned: x} }
func list(x []wire.Value) wire.Value { return wire.Value{Tag: 7, List: x} }
func sortRefs(x []wire.Value) {
	sort.Slice(x, func(i, j int) bool { return x[i].Reference.String() < x[j].Reference.String() })
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
