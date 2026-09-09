package goprovider

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
)

// ResolutionManifest is a provider-level record of decisions already proven
// by go/types. It contains no source bytes or editor revision.
type ResolutionManifest struct{ Packages []ResolvedPackage }
type ResolvedPackage struct {
	Name         string
	Root         bool
	Files        []string
	Declarations []ResolvedDeclaration
	Imports      []ResolvedImport
}
type ResolvedDeclaration struct {
	ID, Name string
	Exported bool
	Location ProjectLocation
}
type ResolvedImport struct {
	Alias, Path, ResolvedPath string
	Local                     bool
	Location                  ProjectLocation
}
type ProjectLocation struct {
	File         string
	Line, Column int
}

func resolveSnapshot(snapshot DocumentSnapshot) (ResolutionManifest, []SessionDiagnostic) {
	units, diagnostics := checkSessionPackages(snapshot)
	if len(diagnostics) != 0 {
		return ResolutionManifest{}, diagnostics
	}
	local := map[string]bool{}
	for _, u := range units {
		local[u.path] = true
	}
	seenIDs := map[string]ProjectLocation{}
	out := ResolutionManifest{}
	for _, u := range units {
		p := ResolvedPackage{Name: u.path, Root: u.path == snapshot.PackagePath}
		importsByPath := map[string]*types.Package{}
		for _, dependency := range u.pkg.Imports() {
			importsByPath[dependency.Path()] = dependency
		}
		for _, file := range u.files {
			fileName := filepath.ToSlash(u.fset.Position(file.Pos()).Filename)
			p.Files = append(p.Files, fileName)
			for _, spec := range file.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err != nil {
					continue
				}
				position := u.fset.Position(spec.Pos())
				alias := ""
				if spec.Name != nil {
					alias = spec.Name.Name
				} else if dependency := importsByPath[path]; dependency != nil {
					alias = dependency.Name()
				}
				p.Imports = append(p.Imports, ResolvedImport{Alias: alias, Path: path, ResolvedPath: path, Local: local[path], Location: location(position.Filename, position.Line, position.Column)})
			}
			for _, item := range file.Decls {
				fn, ok := item.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				object, ok := u.info.Defs[fn.Name].(*types.Func)
				if !ok {
					continue
				}
				declarationID := u.semanticIDs[object]
				if declarationID == "" {
					declarationID = stableID("session-declaration", u.path, fn.Name.Name)
				}
				position := u.fset.Position(fn.Name.Pos())
				where := location(position.Filename, position.Line, position.Column)
				if prior, exists := seenIDs[declarationID]; exists {
					diagnostics = append(diagnostics, SessionDiagnostic{Code: "go.duplicate_semantic_identity", Message: "semantic declaration identity duplicates " + prior.File + ":" + strconv.Itoa(prior.Line) + ":" + strconv.Itoa(prior.Column), File: where.File, Line: where.Line, Column: where.Column, Severity: "error"})
					continue
				}
				seenIDs[declarationID] = where
				p.Declarations = append(p.Declarations, ResolvedDeclaration{ID: declarationID, Name: fn.Name.Name, Exported: ast.IsExported(fn.Name.Name), Location: where})
			}
		}
		sort.Strings(p.Files)
		sort.Slice(p.Declarations, func(i, j int) bool { return p.Declarations[i].ID < p.Declarations[j].ID })
		sort.Slice(p.Imports, func(i, j int) bool {
			a, b := p.Imports[i], p.Imports[j]
			if a.Location.File != b.Location.File {
				return a.Location.File < b.Location.File
			}
			if a.Location.Line != b.Location.Line {
				return a.Location.Line < b.Location.Line
			}
			if a.Location.Column != b.Location.Column {
				return a.Location.Column < b.Location.Column
			}
			if a.Path != b.Path {
				return a.Path < b.Path
			}
			return a.Alias < b.Alias
		})
		out.Packages = append(out.Packages, p)
	}
	sort.Slice(out.Packages, func(i, j int) bool { return out.Packages[i].Name < out.Packages[j].Name })
	return out, sortedDiagnostics(diagnostics)
}
func location(file string, line, column int) ProjectLocation {
	return ProjectLocation{File: filepath.ToSlash(file), Line: line, Column: column}
}
func cloneResolutionManifest(in ResolutionManifest) ResolutionManifest {
	out := ResolutionManifest{Packages: make([]ResolvedPackage, len(in.Packages))}
	for i, p := range in.Packages {
		out.Packages[i] = p
		out.Packages[i].Files = append([]string(nil), p.Files...)
		out.Packages[i].Declarations = append([]ResolvedDeclaration(nil), p.Declarations...)
		out.Packages[i].Imports = append([]ResolvedImport(nil), p.Imports...)
	}
	return out
}
