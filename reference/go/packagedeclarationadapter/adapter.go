// Package packagedeclarationadapter derives language-neutral declaration
// ownership from an authenticated Package-v2 detail graph. It does not inspect
// provider metadata or source-language syntax.
package packagedeclarationadapter

import (
	"fmt"
	"sort"

	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/wire"
)

// DataTypes returns all record declarations and generated generic realizations
// reachable from authenticated Package-v2 members. The historical name is
// retained for callers; no source-language metadata participates.
func DataTypes(raw []byte) ([]packagev3instance.Declaration, error) {
	if err := packagedetailinstance.ValidateV4(raw); err != nil {
		return nil, fmt.Errorf("package_declaration_adapter.base:%w", err)
	}
	e, err := wire.Decode(raw)
	if err != nil {
		return nil, err
	}
	out := []packagev3instance.Declaration{}
	type owner struct {
		detail wire.ID
		origin packagedetail.Origin
	}
	direct := map[wire.ID]owner{}
	candidates := map[wire.ID][]owner{}
	for _, detail := range e.Entities {
		if detail.Schema != id("b021") {
			continue
		}
		for _, value := range detail.Fields[id("b211")].List {
			member := e.Entities[value.Reference]
			declaration := member.Fields[id("b220")].Reference
			origin, er := decodeOrigin(e, member.Fields[id("b224")].Reference)
			if er != nil {
				return nil, er
			}
			visibilityEntity := e.Entities[member.Fields[id("b222")].Reference]
			visibility := packagedetail.Visibility(visibilityEntity.Fields[id("b230")].Unsigned)
			exportName := ""
			if x := member.Fields[id("b223")]; x.Tag == 5 {
				exportName = string(x.Bytes)
			}
			if e.Entities[declaration].Schema == id("9030") {
				direct[declaration] = owner{detail.ID, origin}
				out = append(out, packagev3instance.Declaration{Identity: declaration.String(), OwnerDetail: detail.ID.String(), Origin: origin, Visibility: visibility, Kind: packagev3instance.DataType, Name: string(member.Fields[id("b221")].Bytes), ExportName: exportName})
			}
			for reachable := range closure(e, declaration) {
				if e.Entities[reachable].Schema == id("9042") {
					candidates[reachable] = append(candidates[reachable], owner{detail.ID, origin})
				}
			}
		}
	}
	for declaration, owners := range candidates {
		if _, ok := direct[declaration]; ok {
			continue
		}
		sort.Slice(owners, func(i, j int) bool { return owners[i].detail.String() < owners[j].detail.String() })
		selected := owners[0]
		out = append(out, packagev3instance.Declaration{Identity: declaration.String(), OwnerDetail: selected.detail.String(), Origin: selected.origin, Visibility: packagedetail.Package, Kind: packagev3instance.GenericRealization, Name: declaration.String()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	return out, nil
}

func closure(e wire.Envelope, root wire.ID) map[wire.ID]bool {
	out := map[wire.ID]bool{}
	var visit func(wire.ID)
	visit = func(x wire.ID) {
		if out[x] {
			return
		}
		out[x] = true
		q, ok := e.Entities[x]
		if !ok {
			return
		}
		for _, v := range q.Fields {
			if v.Tag == 6 {
				visit(v.Reference)
			}
			if v.Tag == 7 {
				for _, item := range v.List {
					if item.Tag == 6 {
						visit(item.Reference)
					}
				}
			}
		}
	}
	visit(root)
	return out
}

func decodeOrigin(e wire.Envelope, identity wire.ID) (packagedetail.Origin, error) {
	q, ok := e.Entities[identity]
	if !ok || q.Schema != id("b026") || len(q.Fields[id("b262")].Bytes) != 32 {
		return packagedetail.Origin{}, fmt.Errorf("package_declaration_adapter.origin")
	}
	var digest [32]byte
	copy(digest[:], q.Fields[id("b262")].Bytes)
	return packagedetail.Origin{SourceIdentity: q.Fields[id("b260")].Reference.String(), Path: string(q.Fields[id("b261")].Bytes), ContentDigest: digest, ByteStart: q.Fields[id("b263")].Unsigned, ByteEnd: q.Fields[id("b264")].Unsigned, StartLine: uint32(q.Fields[id("b265")].Unsigned), StartColumn: uint32(q.Fields[id("b266")].Unsigned), EndLine: uint32(q.Fields[id("b267")].Unsigned), EndColumn: uint32(q.Fields[id("b268")].Unsigned)}, nil
}
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
