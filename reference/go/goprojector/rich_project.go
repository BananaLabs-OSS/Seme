package goprojector

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"
)

// ProjectPackagesRich emits one ordinary Go source file per explicitly owned
// package. Ownership is validated before any output is returned.
func ProjectPackagesRich(g1 []byte, ownership RichPackageOwnership) (map[string][]byte, error) {
	plans, err := planRichPackages(g1, ownership)
	if err != nil {
		return nil, err
	}
	whole, err := Project(g1, "projection")
	if err != nil {
		return nil, err
	}
	owned := map[string]OwnedDeclaration{}
	for _, d := range ownership.Declarations {
		owned[d.ID] = d
	}
	family := map[string]OwnedFamily{}
	for _, f := range ownership.Families {
		family[f.Name] = f
	}
	aliases := map[string]string{}
	for _, p := range ownership.Packages {
		aliases[p.Identity] = p.Name
	}
	out := map[string][]byte{}
	for _, pkg := range ownership.Packages {
		fset := token.NewFileSet()
		file, er := parser.ParseFile(fset, "projected.go", whole, parser.ParseComments)
		if er != nil {
			return nil, er
		}
		objectOwner := map[*ast.Object]string{}
		declOwner := map[ast.Decl]string{}
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				id := directiveID(d.Doc)
				if x, ok := owned[id]; ok {
					declOwner[decl] = x.Package
					if d.Name.Obj != nil {
						objectOwner[d.Name.Obj] = x.Package
					}
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					ts, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					id := directiveID(d.Doc)
					owner := ""
					if x, yes := owned[id]; yes {
						owner = x.Package
					} else if x, yes := family[ts.Name.Name]; yes {
						owner = x.Package
					}
					if owner != "" {
						declOwner[decl] = owner
						if ts.Name.Obj != nil {
							objectOwner[ts.Name.Obj] = owner
						}
					}
				}
			}
		}
		chunks := []string{}
		imports := map[string]bool{}
		for _, decl := range file.Decls {
			if declOwner[decl] != pkg.Identity {
				continue
			}
			start := decl.Pos()
			if doc := declDoc(decl); doc != nil {
				start = doc.Pos()
			}
			begin, end := fset.Position(start).Offset, fset.Position(decl.End()).Offset
			raw := string(whole[begin:end])
			repls := []replacement{}
			noRewrite := map[token.Pos]bool{}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.KeyValueExpr:
					if key, ok := x.Key.(*ast.Ident); ok {
						noRewrite[key.Pos()] = true
					}
				case *ast.SelectorExpr:
					noRewrite[x.Sel.Pos()] = true
				}
				return true
			})
			ast.Inspect(decl, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok || id.Obj == nil || noRewrite[id.Pos()] {
					return true
				}
				owner := objectOwner[id.Obj]
				if owner == "" || owner == pkg.Identity {
					return true
				}
				alias := aliases[owner]
				if alias == "" {
					return true
				}
				imports[owner] = true
				pos := fset.Position(id.Pos()).Offset - begin
				repls = append(repls, replacement{pos, pos + len(id.Name), alias + "." + id.Name})
				return true
			})
			sort.Slice(repls, func(i, j int) bool { return repls[i].start > repls[j].start })
			for _, r := range repls {
				raw = raw[:r.start] + r.text + raw[r.end:]
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				s, ok := n.(*ast.SelectorExpr)
				if ok {
					if x, yes := s.X.(*ast.Ident); yes {
						switch x.Name {
						case "bytes", "log", "maps", "slices":
							imports[x.Name] = true
						}
					}
				}
				return true
			})
			chunks = append(chunks, raw)
		}
		for _, dep := range plans[pkg.Identity].imports {
			imports[dep] = true
		}
		var b strings.Builder
		fmt.Fprintf(&b, "package %s\n\n", pkg.Name)
		paths := make([]string, 0, len(imports))
		for x := range imports {
			paths = append(paths, x)
		}
		sort.Strings(paths)
		if len(paths) > 0 {
			b.WriteString("import (\n")
			for _, x := range paths {
				alias := x
				if strings.Contains(x, "/") {
					alias = aliases[x]
					if alias == "" {
						return nil, fmt.Errorf("go_projection.rich_import_alias:%s", x)
					}
				}
				if alias != path.Base(x) {
					fmt.Fprintf(&b, "\t%s %q\n", alias, x)
				} else {
					fmt.Fprintf(&b, "\t%q\n", x)
				}
			}
			b.WriteString(")\n\n")
		}
		for _, chunk := range chunks {
			b.WriteString(chunk)
			b.WriteString("\n\n")
		}
		formatted, er := format.Source([]byte(b.String()))
		if er != nil {
			return nil, fmt.Errorf("go_projection.rich_format:%s:%w", pkg.Identity, er)
		}
		out[pkg.Identity] = formatted
	}
	return out, nil
}

type replacement struct {
	start, end int
	text       string
}

func directiveID(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	for _, c := range doc.List {
		v := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		if strings.HasPrefix(v, "seme:id ") {
			return strings.TrimSpace(strings.TrimPrefix(v, "seme:id "))
		}
	}
	return ""
}
func declDoc(d ast.Decl) *ast.CommentGroup {
	switch x := d.(type) {
	case *ast.FuncDecl:
		return x.Doc
	case *ast.GenDecl:
		return x.Doc
	}
	return nil
}
