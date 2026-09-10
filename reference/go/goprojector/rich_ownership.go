package goprojector

import (
	"fmt"
	"sort"
)

// RichPackageOwnership is an explicit, contract-neutral projection input. It
// supplies declaration ownership that Package Contract v2 does not represent.
// A future Package revision may authenticate this exact information; callers
// must not infer it from declaration names or identity bytes.
type RichPackageOwnership struct {
	Packages     []RichPackage
	Declarations []OwnedDeclaration
	Families     []OwnedFamily
}

type RichPackage struct {
	Identity     string
	Name         string
	Dependencies []string
}

type DeclarationKind string

const (
	FunctionDeclaration  DeclarationKind = "function"
	RecordDeclaration    DeclarationKind = "record"
	InterfaceDeclaration DeclarationKind = "interface"
	MethodDeclaration    DeclarationKind = "method"
)

type OwnedDeclaration struct {
	ID, Package, Name string
	Kind              DeclarationKind
	Exported          bool
}

type OwnedFamily struct{ Kind, Package, Name string }

// ValidateRichPackageOwnership proves that ownership is complete, unique and
// agrees with the canonical graph. It does not make the ownership authentic.
func ValidateRichPackageOwnership(g1 []byte, in RichPackageOwnership) error {
	g, err := parse(g1)
	if err != nil {
		return err
	}
	packages := map[string]RichPackage{}
	names := map[string]bool{}
	prior := ""
	for _, p := range in.Packages {
		if p.Identity == "" || !identifier(p.Name) || p.Identity <= prior || names[p.Name] {
			return fmt.Errorf("go_projection.rich_package")
		}
		prior = p.Identity
		names[p.Name] = true
		seen := map[string]bool{}
		dp := ""
		for _, d := range p.Dependencies {
			if d == p.Identity || d <= dp || seen[d] {
				return fmt.Errorf("go_projection.rich_dependency")
			}
			dp = d
			seen[d] = true
		}
		packages[p.Identity] = p
	}
	for _, p := range in.Packages {
		for _, d := range p.Dependencies {
			if _, ok := packages[d]; !ok {
				return fmt.Errorf("go_projection.rich_dependency_target:%s", d)
			}
		}
	}
	want := map[string]DeclarationKind{}
	canonicalName := map[string]string{}
	for id, e := range g {
		switch e.schema {
		case sFunction:
			want[id] = FunctionDeclaration
			canonicalName[id], err = text(e, "00000000000000000000000000009110")
		case sRecordType:
			want[id] = RecordDeclaration
			canonicalName[id], err = text(e, "00000000000000000000000000009300")
		case sInterfaceType:
			want[id] = InterfaceDeclaration
			canonicalName[id], err = text(e, "000000000000000000000000000a0100")
		case sMethod:
			want[id] = MethodDeclaration
			canonicalName[id], err = text(e, "000000000000000000000000000a0020")
		}
		if err != nil {
			return err
		}
	}
	owners := map[string]OwnedDeclaration{}
	prior = ""
	for _, d := range in.Declarations {
		if d.ID == "" || d.ID <= prior || d.Package == "" || !identifier(d.Name) {
			return fmt.Errorf("go_projection.rich_declaration_order")
		}
		prior = d.ID
		if _, ok := owners[d.ID]; ok {
			return fmt.Errorf("go_projection.rich_declaration_duplicate")
		}
		if _, ok := packages[d.Package]; !ok {
			return fmt.Errorf("go_projection.rich_declaration_package")
		}
		if want[d.ID] != d.Kind || canonicalName[d.ID] != d.Name {
			return fmt.Errorf("go_projection.rich_declaration_mismatch:%s", d.ID)
		}
		owners[d.ID] = d
	}
	if len(owners) != len(want) {
		return fmt.Errorf("go_projection.rich_declaration_incomplete")
	}
	for id := range want {
		if _, ok := owners[id]; !ok {
			return fmt.Errorf("go_projection.rich_declaration_missing:%s", id)
		}
	}
	for _, d := range owners {
		if d.Kind != MethodDeclaration {
			continue
		}
		m := g[d.ID]
		rid, er := ref(m, "000000000000000000000000000a0021")
		if er != nil {
			return er
		}
		receiver := g[rid]
		tid, er := ref(receiver, "000000000000000000000000000a0001")
		if er != nil {
			return er
		}
		owner, ok := owners[tid]
		if !ok || owner.Kind != RecordDeclaration || owner.Package != d.Package {
			return fmt.Errorf("go_projection.rich_method_owner:%s", d.ID)
		}
	}
	for id, d := range owners {
		if d.Kind != FunctionDeclaration && d.Kind != MethodDeclaration {
			continue
		}
		for callee := range calledFunctions(g, id) {
			c, ok := owners[callee]
			if !ok {
				return fmt.Errorf("go_projection.rich_call_owner:%s", callee)
			}
			if c.Package != d.Package {
				if !c.Exported {
					return fmt.Errorf("go_projection.rich_call_private:%s", callee)
				}
				if !contains(packages[d.Package].Dependencies, c.Package) {
					return fmt.Errorf("go_projection.rich_call_dependency:%s", callee)
				}
			}
		}
	}
	required := map[string]bool{}
	if graphHasSchema(g, sResultType) {
		required["result"] = true
	}
	if graphHasSchema(g, sTransitionType) {
		required["transition"] = true
	}
	seenFamilies := map[string]bool{}
	for _, f := range in.Families {
		if !required[f.Kind] || seenFamilies[f.Kind] || packages[f.Package].Identity == "" || !identifier(f.Name) {
			return fmt.Errorf("go_projection.rich_family")
		}
		seenFamilies[f.Kind] = true
	}
	if len(seenFamilies) != len(required) {
		return fmt.Errorf("go_projection.rich_family_incomplete")
	}
	return nil
}

func NormalizeRichPackageOwnership(in *RichPackageOwnership) {
	sort.Slice(in.Packages, func(i, j int) bool { return in.Packages[i].Identity < in.Packages[j].Identity })
	for i := range in.Packages {
		sort.Strings(in.Packages[i].Dependencies)
	}
	sort.Slice(in.Declarations, func(i, j int) bool { return in.Declarations[i].ID < in.Declarations[j].ID })
	sort.Slice(in.Families, func(i, j int) bool { return in.Families[i].Kind < in.Families[j].Kind })
}
func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
