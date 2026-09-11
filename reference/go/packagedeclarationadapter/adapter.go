// Package packagedeclarationadapter derives language-neutral declaration
// ownership from an authenticated Package-v2 detail graph. It does not inspect
// provider metadata or source-language syntax.
package packagedeclarationadapter

import (
	"fmt"

	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/wire"
)

// DataTypes returns every canonical record owned by Package v2. Other richer
// declaration kinds require explicit provider evidence and are not inferred.
func DataTypes(raw []byte) ([]packagev3instance.Declaration, error) {
	if err := packagedetailinstance.ValidateV4(raw); err != nil {
		return nil, fmt.Errorf("package_declaration_adapter.base:%w", err)
	}
	e, err := wire.Decode(raw)
	if err != nil {
		return nil, err
	}
	out := []packagev3instance.Declaration{}
	for _, detail := range e.Entities {
		if detail.Schema != id("b021") {
			continue
		}
		for _, value := range detail.Fields[id("b211")].List {
			member := e.Entities[value.Reference]
			declaration := member.Fields[id("b220")].Reference
			if e.Entities[declaration].Schema != id("9030") {
				continue
			}
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
			out = append(out, packagev3instance.Declaration{Identity: declaration.String(), OwnerDetail: detail.ID.String(), Origin: origin, Visibility: visibility, Kind: packagev3instance.DataType, Name: string(member.Fields[id("b221")].Bytes), ExportName: exportName})
		}
	}
	return out, nil
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
