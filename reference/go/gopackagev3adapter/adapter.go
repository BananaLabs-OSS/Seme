// Package gopackagev3adapter reconciles same-run Go typed evidence with the
// authenticated, language-neutral Package v2 graph. It produces only neutral
// Package v3 declaration inputs and never derives ownership from names.
package gopackagev3adapter

import (
	"fmt"
	"reflect"
	"sort"

	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/wire"
)

func Convert(resolution goprovider.ResolutionManifest, metadata []goprovider.PackageMetadata, packageV2 []byte) ([]packagev3instance.Declaration, error) {
	if err := packagedetailinstance.Validate(packageV2); err != nil {
		return nil, fmt.Errorf("go_package_v3.base:%w", err)
	}
	e, err := wire.Decode(packageV2)
	if err != nil {
		return nil, err
	}
	meta := map[string][]goprovider.SemanticDeclarationMetadata{}
	for _, p := range metadata {
		if _, ok := meta[p.Name]; ok {
			return nil, fmt.Errorf("go_package_v3.metadata_duplicate")
		}
		meta[p.Name] = p.Supplemental
	}
	resolved := map[string][]goprovider.SemanticDeclarationMetadata{}
	for _, p := range resolution.Packages {
		if _, ok := resolved[p.Name]; ok {
			return nil, fmt.Errorf("go_package_v3.resolution_duplicate")
		}
		resolved[p.Name] = p.Supplemental
	}
	if !reflect.DeepEqual(meta, resolved) {
		return nil, fmt.Errorf("go_package_v3.same_run_mismatch")
	}

	type memberEvidence struct{ detail, origin wire.ID }
	members := map[string]memberEvidence{}
	detailByPackage := map[string]wire.ID{}
	bindings := map[wire.ID][]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema != mustID("b021") {
			continue
		}
		pid := q.Fields[mustID("b210")].Reference
		p := e.Entities[pid]
		name := string(p.Fields[mustID("b100")].Bytes)
		if name == "" || detailByPackage[name] != (wire.ID{}) {
			return nil, fmt.Errorf("go_package_v3.detail")
		}
		detailByPackage[name] = x
		for _, v := range q.Fields[mustID("b211")].List {
			m := e.Entities[v.Reference]
			declaration := m.Fields[mustID("b220")].Reference
			members[declaration.String()] = memberEvidence{x, m.Fields[mustID("b224")].Reference}
		}
		for _, v := range q.Fields[mustID("b212")].List {
			bindings[x] = append(bindings[x], v.Reference)
		}
	}

	out := []packagev3instance.Declaration{}
	for packageName, declarations := range meta {
		detail := detailByPackage[packageName]
		if detail == (wire.ID{}) {
			return nil, fmt.Errorf("go_package_v3.package_missing:%s", packageName)
		}
		for _, item := range declarations {
			owned, ok := members[item.Declaration]
			if !ok || owned.detail != detail {
				return nil, fmt.Errorf("go_package_v3.member_owner:%s", item.Declaration)
			}
			if !originMatches(e, owned.origin, item.Origin) {
				return nil, fmt.Errorf("go_package_v3.origin:%s", item.Declaration)
			}
			kind, ok := kind(item.Kind)
			if !ok {
				return nil, fmt.Errorf("go_package_v3.kind")
			}
			refs := []string{}
			for _, requested := range item.ImportReferences {
				binding, er := exactBinding(e, bindings[detail], requested)
				if er != nil {
					return nil, er
				}
				refs = append(refs, binding.String())
			}
			sort.Strings(refs)
			name := item.Name
			if item.Kind == goprovider.SemanticGenericRealization {
				name = item.Declaration
			}
			origin, er := neutralOrigin(e, owned.origin)
			if er != nil {
				return nil, er
			}
			visibility := packagedetail.Package
			if item.Exported {
				visibility = packagedetail.Public
			}
			out = append(out, packagev3instance.Declaration{Identity: item.Declaration, OwnerDetail: detail.String(), Origin: origin, Visibility: visibility, Kind: kind, Name: name, ExportName: item.ExportName, ReferencedImports: refs, GenericDefinition: item.GenericDefinition})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	for i := 1; i < len(out); i++ {
		if out[i-1].Identity == out[i].Identity {
			return nil, fmt.Errorf("go_package_v3.declaration_duplicate")
		}
	}
	return out, nil
}

func kind(k goprovider.SemanticDeclarationKind) (packagev3instance.Kind, bool) {
	switch k {
	case goprovider.SemanticRecord:
		return packagev3instance.DataType, true
	case goprovider.SemanticInterface:
		return packagev3instance.BehavioralInterface, true
	case goprovider.SemanticMethod:
		return packagev3instance.ReceiverCallable, true
	case goprovider.SemanticGenericRealization:
		return packagev3instance.GenericRealization, true
	}
	return 0, false
}

func exactBinding(e wire.Envelope, candidates []wire.ID, want goprovider.SemanticImportReference) (wire.ID, error) {
	var found wire.ID
	for _, x := range candidates {
		q := e.Entities[x]
		if string(q.Fields[mustID("b241")].Bytes) != want.Requested || optionalString(q.Fields[mustID("b240")]) != want.Alias || !originMatches(e, q.Fields[mustID("b245")].Reference, want.Location) {
			continue
		}
		class := e.Entities[q.Fields[mustID("b242")].Reference].Fields[mustID("b250")].Unsigned
		if (class == 0) != want.Local {
			continue
		}
		if want.Local {
			local := q.Fields[mustID("b243")].Reference
			if string(e.Entities[local].Fields[mustID("b100")].Bytes) != want.Resolved {
				continue
			}
		}
		if found != (wire.ID{}) {
			return wire.ID{}, fmt.Errorf("go_package_v3.import_ambiguous")
		}
		found = x
	}
	if found == (wire.ID{}) {
		return wire.ID{}, fmt.Errorf("go_package_v3.import_missing:%s", want.Requested)
	}
	return found, nil
}

func originMatches(e wire.Envelope, x wire.ID, want goprovider.ProjectLocation) bool {
	q, ok := e.Entities[x]
	if !ok || q.Schema != mustID("b026") {
		return false
	}
	return string(q.Fields[mustID("b261")].Bytes) == want.File && q.Fields[mustID("b263")].Unsigned == uint64(want.ByteStart) && q.Fields[mustID("b264")].Unsigned == uint64(want.ByteEnd) && q.Fields[mustID("b265")].Unsigned == uint64(want.Line) && q.Fields[mustID("b266")].Unsigned == uint64(want.Column) && q.Fields[mustID("b267")].Unsigned == uint64(want.EndLine) && q.Fields[mustID("b268")].Unsigned == uint64(want.EndColumn) && len(q.Fields[mustID("b262")].Bytes) == 32
}
func neutralOrigin(e wire.Envelope, x wire.ID) (packagedetail.Origin, error) {
	q, ok := e.Entities[x]
	if !ok || q.Schema != mustID("b026") {
		return packagedetail.Origin{}, fmt.Errorf("go_package_v3.origin_entity")
	}
	uid := q.Fields[mustID("b260")].Reference
	var digest [32]byte
	raw := q.Fields[mustID("b262")].Bytes
	if len(raw) != 32 {
		return packagedetail.Origin{}, fmt.Errorf("go_package_v3.origin_digest")
	}
	copy(digest[:], raw)
	return packagedetail.Origin{SourceIdentity: uid.String(), Path: string(q.Fields[mustID("b261")].Bytes), ContentDigest: digest, ByteStart: q.Fields[mustID("b263")].Unsigned, ByteEnd: q.Fields[mustID("b264")].Unsigned, StartLine: uint32(q.Fields[mustID("b265")].Unsigned), StartColumn: uint32(q.Fields[mustID("b266")].Unsigned), EndLine: uint32(q.Fields[mustID("b267")].Unsigned), EndColumn: uint32(q.Fields[mustID("b268")].Unsigned)}, nil
}
func optionalString(v wire.Value) string {
	if v.Tag == 5 {
		return string(v.Bytes)
	}
	return ""
}
func mustID(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
