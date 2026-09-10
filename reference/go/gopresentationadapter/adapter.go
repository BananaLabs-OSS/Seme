// Package gopresentationadapter resolves typed Go alias metadata against one
// authenticated Project-v9 graph without treating Go names as semantic types.
package gopresentationadapter

import (
	"fmt"
	"sort"

	"seme.local/reference/goprovider"
	"seme.local/reference/presentationinstance"
	"seme.local/reference/projectv9instance"
	"seme.local/reference/wire"
)

func Resolve(project projectv9instance.Inputs, packages []goprovider.PackageMetadata) (presentationinstance.Model, error) {
	if err := projectv9instance.Validate(project); err != nil {
		return presentationinstance.Model{}, fmt.Errorf("go_presentation.project:%w", err)
	}
	e, err := wire.Decode(project.Composed)
	if err != nil {
		return presentationinstance.Model{}, err
	}
	owners := map[string]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			n := string(q.Fields[id("b100")].Bytes)
			if n == "" || owners[n] != (wire.ID{}) {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.owner")
			}
			owners[n] = x
		}
	}
	units := map[string]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("e015") {
			p := string(q.Fields[id("e150")].Bytes)
			if p == "" || units[p] != (wire.ID{}) {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.unit")
			}
			units[p] = x
		}
	}
	bindings := map[wire.ID]map[string]map[string]wire.ID{}
	for _, q := range e.Entities {
		if q.Schema != id("b021") {
			continue
		}
		owner := q.Fields[id("b210")].Reference
		if bindings[owner] == nil {
			bindings[owner] = map[string]map[string]wire.ID{}
		}
		for _, v := range q.Fields[id("b212")].List {
			b := e.Entities[v.Reference]
			if b.Schema != id("b024") {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.binding_schema:%s", v.Reference)
			}
			requested := string(b.Fields[id("b241")].Bytes)
			origin := e.Entities[b.Fields[id("b245")].Reference]
			document := string(origin.Fields[id("b261")].Bytes)
			if requested == "" {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.binding_name:%s", v.Reference)
			}
			if origin.Schema != id("b026") || document == "" {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.binding_origin:%s", v.Reference)
			}
			if bindings[owner][document] == nil {
				bindings[owner][document] = map[string]wire.ID{}
			}
			if bindings[owner][document][requested] != (wire.ID{}) {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.binding_duplicate:%s:%s", document, requested)
			}
			bindings[owner][document][requested] = v.Reference
		}
	}
	var out presentationinstance.Model
	seen := map[string]bool{}
	for _, p := range packages {
		owner := owners[p.Name]
		if owner == (wire.ID{}) {
			return presentationinstance.Model{}, fmt.Errorf("go_presentation.package:%s", p.Name)
		}
		for _, a := range p.Aliases {
			if a.Package != p.Name || a.Generic || seen[p.Name+"\x00"+a.Name] {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.alias:%s", a.Name)
			}
			seen[p.Name+"\x00"+a.Name] = true
			target, er := wire.ParseID(a.Target)
			if er != nil || e.Entities[target].ID == (wire.ID{}) {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.target:%s", a.Name)
			}
			unit := units[a.Document]
			q := e.Entities[unit]
			digest := q.Fields[id("e151")]
			if unit == (wire.ID{}) || digest.Tag != 5 || len(digest.Bytes) != 32 {
				return presentationinstance.Model{}, fmt.Errorf("go_presentation.source:%s", a.Name)
			}
			var sum [32]byte
			copy(sum[:], digest.Bytes)
			refs := make([]wire.ID, 0, len(a.ReferencedImports))
			seenImports := map[string]bool{}
			for _, path := range a.ReferencedImports {
				if seenImports[path] {
					return presentationinstance.Model{}, fmt.Errorf("go_presentation.import_duplicate:%s:%s", a.Name, path)
				}
				seenImports[path] = true
				x := bindings[owner][a.Document][path]
				if x == (wire.ID{}) {
					return presentationinstance.Model{}, fmt.Errorf("go_presentation.import:%s:%s", a.Name, path)
				}
				refs = append(refs, x)
			}
			sort.Slice(refs, func(i, j int) bool { return refs[i].String() < refs[j].String() })
			visibility := uint64(0)
			if a.Exported {
				visibility = 1
			}
			out.Aliases = append(out.Aliases, presentationinstance.Alias{Owner: owner, Target: target, SourceUnit: unit, Name: a.Name, Visibility: visibility, Path: a.Document, Digest: sum, Start: uint64(a.Start), End: uint64(a.End), StartLine: uint64(a.Line), StartColumn: uint64(a.Column), EndLine: uint64(a.EndLine), EndColumn: uint64(a.EndColumn), ImportBindings: refs})
		}
	}
	return out, nil
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
