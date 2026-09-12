package goprojector

import (
	"fmt"
	"path"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/wire"
)

// ProjectPackagesV3 derives projection ownership exclusively from an
// authenticated Package v3 instance. No caller-supplied ownership participates.
func ProjectPackagesV3(g1 []byte, contracts contractcatalog.ProjectContractSetV5, packageV2, packageV3 []byte) (map[string][]byte, error) {
	if err := packagev3instance.Validate(contracts, packageV2, packageV3); err != nil {
		return nil, fmt.Errorf("go_projection.package_v3:%w", err)
	}
	e, err := wire.Decode(packageV3)
	if err != nil {
		return nil, err
	}
	ownership, err := richOwnershipV3(e)
	if err != nil {
		return nil, err
	}
	// Wire entity storage is intentionally unordered. Normalize the derived
	// ownership model before applying the projector's strict order checks.
	NormalizeRichPackageOwnership(&ownership)
	out, err := ProjectPackagesRich(g1, ownership)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ProjectPackagesV4 derives projection ownership from a Package-v4 artifact
// authenticated by the one-run Execution-v36/Project-v8 contract set.
func ProjectPackagesV4(g1 []byte, contracts contractcatalog.ProjectContractSetV8, packageV2, packageV4 []byte) (map[string][]byte, error) {
	return ProjectPackagesV4WithAliases(g1, contracts, packageV2, packageV4, nil)
}

// OwnershipV4 returns the complete language-neutral ownership authenticated by
// Package-v4. Native project layout is deliberately not part of this model.
func OwnershipV4(contracts contractcatalog.ProjectContractSetV8, packageV2, packageV4 []byte) (RichPackageOwnership, error) {
	if err := packagev3instance.ValidateV4(contracts, packageV2, packageV4); err != nil {
		return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v4:%w", err)
	}
	e, err := wire.Decode(packageV4)
	if err != nil {
		return RichPackageOwnership{}, err
	}
	ownership, err := richOwnershipV3(e)
	if err != nil {
		return RichPackageOwnership{}, err
	}
	NormalizeRichPackageOwnership(&ownership)
	return ownership, nil
}

func ProjectPackagesV4WithAliases(g1 []byte, contracts contractcatalog.ProjectContractSetV8, packageV2, packageV4 []byte, aliases []AliasPresentation) (map[string][]byte, error) {
	if err := packagev3instance.ValidateV4(contracts, packageV2, packageV4); err != nil {
		return nil, fmt.Errorf("go_projection.package_v4:%w", err)
	}
	e, err := wire.Decode(packageV4)
	if err != nil {
		return nil, err
	}
	ownership, err := richOwnershipV3(e)
	if err != nil {
		return nil, err
	}
	NormalizeRichPackageOwnership(&ownership)
	return ProjectPackagesRichWithAliases(g1, ownership, aliases)
}

func richOwnershipV3(e wire.Envelope) (RichPackageOwnership, error) {
	graphIDs := withWireSchema(e, wid("b020"))
	complete := withWireSchema(e, wid("b029"))
	if len(graphIDs) != 1 || len(complete) != 1 {
		return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_graph")
	}
	packageNames := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == wid("b010") {
			v := q.Fields[wid("b100")]
			if v.Tag != 5 || len(v.Bytes) == 0 {
				return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_package")
			}
			packageNames[x] = string(v.Bytes)
		}
	}
	detailPackage := map[wire.ID]wire.ID{}
	out := RichPackageOwnership{}
	for _, dv := range e.Entities[graphIDs[0]].Fields[wid("b200")].List {
		if dv.Tag != 6 {
			return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_detail_ref")
		}
		d := e.Entities[dv.Reference]
		pid := d.Fields[wid("b210")].Reference
		identity := packageNames[pid]
		name := path.Base(identity)
		if identity == "" || !identifier(name) {
			return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_package_identity")
		}
		detailPackage[d.ID] = pid
		p := RichPackage{Identity: identity, Name: name}
		dependencies := map[string]bool{}
		for _, iv := range d.Fields[wid("b212")].List {
			binding := e.Entities[iv.Reference]
			class := e.Entities[binding.Fields[wid("b242")].Reference].Fields[wid("b250")].Unsigned
			if class == 0 {
				target := binding.Fields[wid("b243")]
				if target.Tag != 6 || packageNames[target.Reference] == "" {
					return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_dependency")
				}
				dependencies[packageNames[target.Reference]] = true
			}
		}
		for dependency := range dependencies {
			p.Dependencies = append(p.Dependencies, dependency)
		}
		sort.Strings(p.Dependencies)
		for _, mv := range d.Fields[wid("b211")].List {
			m := e.Entities[mv.Reference]
			decl := m.Fields[wid("b220")].Reference
			// Package v2 now annotates both callable interfaces and other
			// declarations. Only top-level Function entities belong to this
			// legacy callable loop; Package v3 owns the richer kinds below.
			if e.Entities[decl].Schema != wid("9011") {
				continue
			}
			namev := m.Fields[wid("b221")]
			if namev.Tag != 5 {
				return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_member")
			}
			_, exported := m.Fields[wid("b223")]
			out.Declarations = append(out.Declarations, OwnedDeclaration{ID: decl.String(), Package: identity, Name: string(namev.Bytes), Kind: FunctionDeclaration, Exported: exported})
		}
		out.Packages = append(out.Packages, p)
	}
	families := map[string]OwnedFamily{}
	items := e.Entities[complete[0]].Fields[wid("b291")].List
	for _, v := range items {
		q := e.Entities[v.Reference]
		decl := q.Fields[wid("b280")].Reference
		ownerDetail := q.Fields[wid("b281")].Reference
		pid := detailPackage[ownerDetail]
		owner := packageNames[pid]
		if owner == "" {
			return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_owner")
		}
		kindCode := e.Entities[q.Fields[wid("b282")].Reference].Fields[wid("b270")].Unsigned
		name := string(q.Fields[wid("b283")].Bytes)
		_, exported := q.Fields[wid("b285")]
		switch kindCode {
		case 0:
			out.Declarations = append(out.Declarations, OwnedDeclaration{ID: decl.String(), Package: owner, Name: name, Kind: RecordDeclaration, Exported: exported})
		case 1:
			out.Declarations = append(out.Declarations, OwnedDeclaration{ID: decl.String(), Package: owner, Name: name, Kind: InterfaceDeclaration, Exported: exported})
		case 2:
			out.Declarations = append(out.Declarations, OwnedDeclaration{ID: decl.String(), Package: owner, Name: name, Kind: MethodDeclaration, Exported: exported})
		case 4:
			schema := e.Entities[decl].Schema
			fk, fn := "", ""
			switch schema {
			case wid("9042"):
				fk, fn = "result", "Result"
			case wid("a004"):
				fk, fn = "transition", "Transition"
			case wid("a050"):
				fk, fn = "option", "Option"
			default:
				return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_family_schema")
			}
			if prior, ok := families[fk]; ok && (prior.Package != owner || prior.Name != fn) {
				return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_family_owner")
			}
			families[fk] = OwnedFamily{Kind: fk, Package: owner, Name: fn}
		default:
			return RichPackageOwnership{}, fmt.Errorf("go_projection.package_v3_kind")
		}
	}
	for _, f := range families {
		out.Families = append(out.Families, f)
	}
	NormalizeRichPackageOwnership(&out)
	return out, nil
}

func withWireSchema(e wire.Envelope, s wire.ID) []wire.ID {
	out := []wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == s {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}
func wid(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
