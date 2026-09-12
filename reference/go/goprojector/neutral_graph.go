package goprojector

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"seme.local/reference/contractcatalog"
)

// ProjectionGraph is the source-layout-neutral ownership view consumed by the
// JavaScript and Lua multi-module projectors. Paths and import spellings are
// target presentation, while identities and ownership remain canonical.
type ProjectionGraph struct {
	Packages []ProjectionPackage `json:"Packages"`
}
type ProjectionPackage struct {
	Identity string             `json:"Identity"`
	Root     bool               `json:"Root"`
	Sources  []ProjectionSource `json:"Sources"`
	Members  []ProjectionMember `json:"Members"`
	Imports  []ProjectionImport `json:"Imports"`
	Effects  []string           `json:"Effects"`
}
type ProjectionSource struct {
	Path string `json:"Path"`
}
type ProjectionMember struct {
	Identity   string `json:"Identity"`
	Name       string `json:"Name"`
	ExportName string `json:"ExportName"`
	Visibility uint64 `json:"Visibility"`
	Callable   bool   `json:"Callable"`
}
type ProjectionImport struct {
	Alias     string `json:"Alias"`
	Requested string `json:"Requested"`
	Resolved  string `json:"Resolved"`
}

// ProjectionGraphV4 derives a native module layout from authenticated
// Package-v4 ownership without consulting original source provenance.
func ProjectionGraphV4(g1 []byte, contracts contractcatalog.ProjectContractSetV8, packageV2, packageV4 []byte, module, language string) (ProjectionGraph, error) {
	if language != "javascript" && language != "lua" {
		return ProjectionGraph{}, fmt.Errorf("go_projection.graph_language")
	}
	ownership, err := OwnershipV4(contracts, packageV2, packageV4)
	if err != nil {
		return ProjectionGraph{}, err
	}
	if err = ValidateRichPackageOwnership(g1, ownership); err != nil {
		return ProjectionGraph{}, err
	}
	entities, err := parse(g1)
	if err != nil {
		return ProjectionGraph{}, err
	}
	owners := map[string]OwnedDeclaration{}
	for _, d := range ownership.Declarations {
		owners[d.ID] = d
	}
	byPackage := map[string][]OwnedDeclaration{}
	for _, d := range ownership.Declarations {
		if d.Kind != FunctionDeclaration && d.Kind != RecordDeclaration {
			return ProjectionGraph{}, fmt.Errorf("go_projection.graph_declaration:%s", d.Kind)
		}
		byPackage[d.Package] = append(byPackage[d.Package], d)
	}
	files := map[string]string{}
	for _, p := range ownership.Packages {
		relative, ok := projectionRelative(module, p.Identity)
		if !ok {
			return ProjectionGraph{}, fmt.Errorf("go_projection.graph_package:%s", p.Identity)
		}
		if relative == "." {
			relative = "application"
		}
		ext := ".js"
		if language == "lua" {
			ext = ".lua"
		}
		files[p.Identity] = relative + ext
	}
	var out ProjectionGraph
	for _, p := range ownership.Packages {
		item := ProjectionPackage{Identity: p.Identity, Root: p.Root, Sources: []ProjectionSource{{Path: files[p.Identity]}}, Members: []ProjectionMember{}, Imports: []ProjectionImport{}, Effects: []string{}}
		imports := map[string]ProjectionImport{}
		for _, d := range byPackage[p.Identity] {
			item.Members = append(item.Members, ProjectionMember{Identity: d.ID, Name: d.Name, ExportName: func() string {
				if d.Exported {
					return d.Name
				}
				return ""
			}(), Visibility: func() uint64 {
				if d.Exported {
					return 2
				}
				return 0
			}(), Callable: d.Kind == FunctionDeclaration})
			if d.Kind != FunctionDeclaration {
				continue
			}
			for callee := range calledFunctions(entities, d.ID) {
				target := owners[callee]
				if target.Package == p.Identity {
					continue
				}
				if !target.Exported {
					return ProjectionGraph{}, fmt.Errorf("go_projection.graph_private:%s", callee)
				}
				requested := projectionImport(files[p.Identity], files[target.Package], language)
				key := target.Package + "\x00" + target.Name
				if old, ok := imports[key]; ok && old.Requested != requested {
					return ProjectionGraph{}, fmt.Errorf("go_projection.graph_import")
				}
				imports[key] = ProjectionImport{Alias: target.Name, Requested: requested, Resolved: target.Package}
			}
		}
		for _, value := range imports {
			item.Imports = append(item.Imports, value)
		}
		sort.Slice(item.Members, func(i, j int) bool { return item.Members[i].Identity < item.Members[j].Identity })
		sort.Slice(item.Imports, func(i, j int) bool {
			a, b := item.Imports[i], item.Imports[j]
			return a.Resolved < b.Resolved || a.Resolved == b.Resolved && a.Alias < b.Alias
		})
		out.Packages = append(out.Packages, item)
	}
	if count := func() int {
		n := 0
		for _, p := range out.Packages {
			if p.Root {
				n++
			}
		}
		return n
	}(); count != 1 {
		return ProjectionGraph{}, fmt.Errorf("go_projection.graph_root")
	}
	return out, nil
}
func projectionRelative(module, identity string) (string, bool) {
	if identity == module {
		return ".", true
	}
	prefix := module + "/"
	if !strings.HasPrefix(identity, prefix) {
		return "", false
	}
	relative := strings.TrimPrefix(identity, prefix)
	if relative == "" || path.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, "../") {
		return "", false
	}
	return relative, true
}
func projectionImport(from, to, language string) string {
	target := strings.TrimSuffix(to, path.Ext(to))
	if language == "lua" {
		return strings.ReplaceAll(target, "/", ".")
	}
	relative, err := filepath.Rel(path.Dir(from), to)
	if err != nil {
		panic(err)
	}
	relative = filepath.ToSlash(relative)
	if !strings.HasPrefix(relative, ".") {
		relative = "./" + relative
	}
	return relative
}
