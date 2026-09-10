package goprovider

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"
)

// attachSemanticOwnership derives Package-v3 supplemental ownership directly
// from the checked declarations that produced instances. It deliberately
// refuses to publish an identity unless that identity exists in CanonicalG1's
// construction set.
func attachSemanticOwnership(packages []PackageMetadata, units []*checkedSessionPackage, functions []sessionFunction, instances []graphEntity) *SessionDiagnostic {
	emitted := make(map[string]bool, len(instances))
	for _, entity := range instances {
		emitted[entity.id] = true
	}
	byPackage := make(map[string]*PackageMetadata, len(packages))
	unitByPackage := make(map[string]*checkedSessionPackage, len(units))
	localPackages := make(map[string]bool, len(units))
	for _, unit := range units {
		unitByPackage[unit.path] = unit
		localPackages[unit.path] = true
	}
	for i := range packages {
		byPackage[packages[i].Name] = &packages[i]
	}
	seen := map[string]bool{}
	add := func(item SemanticDeclarationMetadata) *SessionDiagnostic {
		if item.Declaration == "" || !emitted[item.Declaration] || byPackage[item.Package] == nil || seen[item.Declaration] {
			return &SessionDiagnostic{Code: "session.semantic_ownership", Message: "supplemental declaration does not uniquely identify an emitted canonical entity", File: item.Origin.File, Line: item.Origin.Line, Column: item.Origin.Column, Severity: "error"}
		}
		seen[item.Declaration] = true
		byPackage[item.Package].Supplemental = append(byPackage[item.Package].Supplemental, item)
		return nil
	}

	for _, unit := range units {
		for _, file := range unit.files {
			for _, declaration := range file.Decls {
				switch node := declaration.(type) {
				case *ast.GenDecl:
					for _, spec := range node.Specs {
						typeSpec, ok := spec.(*ast.TypeSpec)
						if !ok {
							continue
						}
						object, ok := unit.info.Defs[typeSpec.Name].(*types.TypeName)
						if !ok {
							continue
						}
						named, ok := object.Type().(*types.Named)
						if !ok {
							continue
						}
						kind := SemanticRecord
						prefix := "record"
						switch named.Underlying().(type) {
						case *types.Interface:
							kind, prefix = SemanticInterface, "interface"
						case *types.Struct:
						default:
							continue
						}
						id := unit.semanticIDs[object]
						if id == "" {
							id = stableID("execution", prefix, unit.path, object.Name())
						}
						if !emitted[id] { // Unsupported generic definitions are not invented.
							continue
						}
						paths, refs := referencedImports(unit, typeSpec, unit.path, localPackages)
						if d := add(SemanticDeclarationMetadata{Declaration: id, Package: unit.path, Name: object.Name(), Kind: kind, Exported: object.Exported(), Origin: nodeLocation(unit.fset, typeSpec.Name), ReferencedImports: paths, ImportReferences: refs}); d != nil {
							return d
						}
					}
				}
			}
		}
	}

	for _, function := range functions {
		if function.method && emitted[function.id] {
			paths, refs := referencedImports(unitByPackage[function.packagePath], function.fn, function.packagePath, localPackages)
			if d := add(SemanticDeclarationMetadata{Declaration: function.id, Package: function.packagePath, Name: function.name, Kind: SemanticMethod, Exported: ast.IsExported(function.name), Origin: nodeLocation(function.fset, function.fn.Name), ReferencedImports: paths, ImportReferences: refs}); d != nil {
				return d
			}
		}
		var genericDiagnostic *SessionDiagnostic
		visitSignatureNamed(function.sig, func(named *types.Named) {
			if genericDiagnostic != nil {
				return
			}
			if d := genericRealization(named, emitted); d != nil && !seen[d.Declaration] {
				// The realization belongs to the package declaring its generic
				// family. GenericDefinition remains empty because Execution v35
				// does not emit a corresponding generic-definition entity.
				d.Origin = typeObjectLocation(units, named.Origin().Obj())
				genericDiagnostic = add(*d)
			}
		})
		if genericDiagnostic != nil {
			return genericDiagnostic
		}
	}
	for i := range packages {
		sort.Slice(packages[i].Supplemental, func(a, b int) bool {
			return packages[i].Supplemental[a].Declaration < packages[i].Supplemental[b].Declaration
		})
	}
	return nil
}

func genericRealization(named *types.Named, emitted map[string]bool) *SemanticDeclarationMetadata {
	if named == nil || named.Obj() == nil || named.Obj().Pkg() == nil || named.TypeArgs() == nil || named.TypeArgs().Len() == 0 {
		return nil
	}
	id := goSemanticTypeIdentity(named)
	if !emitted[id] {
		return nil
	}
	return &SemanticDeclarationMetadata{Declaration: id, Package: named.Obj().Pkg().Path(), Name: types.TypeString(named, func(p *types.Package) string { return p.Path() }), Kind: SemanticGenericRealization, Exported: named.Obj().Exported()}
}

func visitSignatureNamed(signature *types.Signature, visit func(*types.Named)) {
	if signature == nil {
		return
	}
	seen := map[types.Type]bool{}
	var walk func(types.Type)
	walk = func(value types.Type) {
		value = types.Unalias(value)
		if value == nil || seen[value] {
			return
		}
		seen[value] = true
		switch t := value.(type) {
		case *types.Named:
			visit(t)
			for i := 0; i < t.TypeArgs().Len(); i++ {
				walk(t.TypeArgs().At(i))
			}
		case *types.Pointer:
			walk(t.Elem())
		case *types.Slice:
			walk(t.Elem())
		case *types.Map:
			walk(t.Key())
			walk(t.Elem())
		case *types.Signature:
			if t.Recv() != nil {
				walk(t.Recv().Type())
			}
			for i := 0; i < t.Params().Len(); i++ {
				walk(t.Params().At(i).Type())
			}
			for i := 0; i < t.Results().Len(); i++ {
				walk(t.Results().At(i).Type())
			}
		}
	}
	walk(signature)
}

func nodeLocation(fset *token.FileSet, node ast.Node) ProjectLocation {
	return locationSpan(fset.Position(node.Pos()), fset.Position(node.End()))
}

func referencedImports(unit *checkedSessionPackage, node ast.Node, own string, local map[string]bool) ([]string, []SemanticImportReference) {
	if unit == nil {
		return nil, nil
	}
	set := map[string]bool{}
	usedFiles := map[string]bool{}
	for expression, object := range unit.info.Uses {
		if expression.Pos() < node.Pos() || expression.Pos() >= node.End() || object == nil || object.Pkg() == nil || object.Pkg().Path() == own {
			continue
		}
		set[object.Pkg().Path()] = true
		usedFiles[unit.fset.Position(expression.Pos()).Filename] = true
	}
	result := make([]string, 0, len(set))
	for path := range set {
		result = append(result, path)
	}
	sort.Strings(result)
	refs := []SemanticImportReference{}
	packageNames := map[string]string{}
	for _, dependency := range unit.pkg.Imports() {
		packageNames[dependency.Path()] = dependency.Name()
	}
	for _, file := range unit.files {
		if !usedFiles[unit.fset.Position(file.Pos()).Filename] {
			continue
		}
		for _, spec := range file.Imports {
			requested, err := strconv.Unquote(spec.Path.Value)
			if err == nil && set[requested] {
				alias := packageNames[requested]
				if spec.Name != nil {
					alias = spec.Name.Name
				}
				refs = append(refs, SemanticImportReference{Alias: alias, Requested: requested, Resolved: requested, Local: local[requested], Location: nodeLocation(unit.fset, spec)})
			}
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		a, b := refs[i], refs[j]
		if a.Requested != b.Requested {
			return a.Requested < b.Requested
		}
		if a.Location.File != b.Location.File {
			return a.Location.File < b.Location.File
		}
		if a.Location.Line != b.Location.Line {
			return a.Location.Line < b.Location.Line
		}
		return a.Location.Column < b.Location.Column
	})
	return result, refs
}

func typeObjectLocation(units []*checkedSessionPackage, object *types.TypeName) ProjectLocation {
	if object == nil {
		return ProjectLocation{}
	}
	for _, unit := range units {
		if object.Pkg() != nil && unit.path == object.Pkg().Path() {
			for identifier, defined := range unit.info.Defs {
				if defined == object {
					return nodeLocation(unit.fset, identifier)
				}
			}
		}
	}
	return ProjectLocation{}
}
