// Package packagedetailmetadata extracts projection metadata only from a
// validated canonical Package Contract v2 instance. It has no v1 fallback.
package packagedetailmetadata

import (
	"fmt"
	"sort"

	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
	"seme.local/reference/packagedetailinstance"
	"seme.local/reference/wire"
)

type Result struct{ Packages []Package }
type Package struct {
	Projection goprovider.PackageMetadata
	Members    []Member
	Imports    []Import
}
type Member struct {
	Projection goprovider.PackageFunctionMetadata
	Visibility packagedetail.Visibility
	ExportName string
}
type Import struct {
	Alias, Requested, Resolved string
	Class                      packagedetail.ImportClass
	Document                   string
	Line, Column               int
}

func Extract(source []byte) (Result, error) {
	if err := packagedetailinstance.Validate(source); err != nil {
		return Result{}, err
	}
	e, err := wire.Decode(source)
	if err != nil {
		return Result{}, err
	}
	root := wire.ID{}
	for _, q := range e.Entities {
		if q.Schema == id("e011") {
			root = q.Fields[id("e113")].Reference
		}
	}
	packageNames := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			packageNames[x] = string(q.Fields[id("b100")].Bytes)
		}
	}
	graphs := withSchema(e, id("b020"))
	if len(graphs) != 1 {
		return Result{}, fmt.Errorf("package_detail_metadata.graph")
	}
	var out Result
	for _, dv := range e.Entities[graphs[0]].Fields[id("b200")].List {
		d := e.Entities[dv.Reference]
		pid := d.Fields[id("b210")].Reference
		name, ok := packageNames[pid]
		if !ok {
			return Result{}, fmt.Errorf("package_detail_metadata.package")
		}
		p := Package{Projection: goprovider.PackageMetadata{Name: name, Root: pid == root, Dependencies: []string{}, Members: []goprovider.PackageFunctionMetadata{}, Functions: []goprovider.PackageFunctionMetadata{}}, Members: []Member{}, Imports: []Import{}}
		for _, mv := range d.Fields[id("b211")].List {
			m := e.Entities[mv.Reference]
			decl := m.Fields[id("b220")].Reference
			fn := e.Entities[decl]
			origin := e.Entities[m.Fields[id("b224")].Reference]
			visibilityEntity := e.Entities[m.Fields[id("b222")].Reference]
			visibility := packagedetail.Visibility(visibilityEntity.Fields[id("b230")].Unsigned)
			projection := goprovider.PackageFunctionMetadata{ID: decl.String(), Name: string(m.Fields[id("b221")].Bytes), Exported: m.Fields[id("b223")].Tag == 5, Document: string(origin.Fields[id("b261")].Bytes), Line: int(origin.Fields[id("b265")].Unsigned), Column: int(origin.Fields[id("b266")].Unsigned)}
			for _, pv := range fn.Fields[id("9111")].List {
				parameter := e.Entities[pv.Reference]
				projection.Parameters = append(projection.Parameters, parameter.Fields[id("9121")].Reference.String())
			}
			projection.Result = fn.Fields[id("9112")].Reference.String()
			exportName := ""
			if v, yes := m.Fields[id("b223")]; yes {
				exportName = string(v.Bytes)
			}
			p.Members = append(p.Members, Member{Projection: projection, Visibility: visibility, ExportName: exportName})
			p.Projection.Members = append(p.Projection.Members, projection)
			if projection.Exported {
				p.Projection.Functions = append(p.Projection.Functions, projection)
			}
		}
		for _, iv := range d.Fields[id("b212")].List {
			im := e.Entities[iv.Reference]
			origin := e.Entities[im.Fields[id("b245")].Reference]
			classEntity := e.Entities[im.Fields[id("b242")].Reference]
			class := packagedetail.ImportClass(classEntity.Fields[id("b250")].Unsigned)
			resolved := ""
			if class == packagedetail.Local {
				resolved = packageNames[im.Fields[id("b243")].Reference]
				p.Projection.Dependencies = append(p.Projection.Dependencies, resolved)
			} else {
				resolved = im.Fields[id("b244")].Reference.String()
			}
			alias := ""
			if v, yes := im.Fields[id("b240")]; yes {
				alias = string(v.Bytes)
			}
			p.Imports = append(p.Imports, Import{Alias: alias, Requested: string(im.Fields[id("b241")].Bytes), Resolved: resolved, Class: class, Document: string(origin.Fields[id("b261")].Bytes), Line: int(origin.Fields[id("b265")].Unsigned), Column: int(origin.Fields[id("b266")].Unsigned)})
		}
		sort.Strings(p.Projection.Dependencies)
		sort.Slice(p.Members, func(i, j int) bool { return p.Members[i].Projection.ID < p.Members[j].Projection.ID })
		sort.Slice(p.Imports, func(i, j int) bool {
			a, b := p.Imports[i], p.Imports[j]
			return a.Document < b.Document || a.Document == b.Document && (a.Line < b.Line || a.Line == b.Line && a.Column < b.Column)
		})
		out.Packages = append(out.Packages, p)
	}
	sort.Slice(out.Packages, func(i, j int) bool { return out.Packages[i].Projection.Name < out.Packages[j].Projection.Name })
	return out, nil
}
func withSchema(e wire.Envelope, s wire.ID) []wire.ID {
	var out []wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			out = append(out, x)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
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
