// Package gopackageadapter strictly reconciles the Go provider's two typed
// views into packagedetail. Full byte provenance is supplied independently.
package gopackageadapter

import (
	"fmt"
	"sort"

	"seme.local/reference/goprovider"
	"seme.local/reference/packagedetail"
)

type Evidence struct {
	Sources map[string]packagedetail.Source
	Origins map[Key]packagedetail.Origin
}
type Key struct {
	File         string
	Line, Column int
}

func Convert(r goprovider.ResolutionManifest, metadata []goprovider.PackageMetadata, evidence Evidence) (packagedetail.Graph, error) {
	meta := map[string]goprovider.PackageMetadata{}
	for _, p := range metadata {
		if _, ok := meta[p.Name]; ok {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.metadata_duplicate")
		}
		meta[p.Name] = p
	}
	g := packagedetail.Graph{}
	seenPackages := map[string]bool{}
	wantedSources := map[string]bool{}
	for _, rp := range r.Packages {
		for _, file := range rp.Files {
			wantedSources[file] = true
		}
	}
	if len(wantedSources) != len(evidence.Sources) {
		return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.source_set")
	}
	for file, source := range evidence.Sources {
		if !wantedSources[file] || source.Path != file {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.source_set")
		}
	}
	for _, rp := range r.Packages {
		if seenPackages[rp.Name] {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.resolution_package_duplicate")
		}
		seenPackages[rp.Name] = true
		mp, ok := meta[rp.Name]
		if !ok {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.metadata_missing")
		}
		delete(meta, rp.Name)
		if rp.Root != mp.Root {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.root_mismatch")
		}
		p := packagedetail.Detail{Identity: rp.Name, Root: rp.Root}
		for _, path := range rp.Files {
			source, ok := evidence.Sources[path]
			if !ok {
				return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.source_missing")
			}
			p.Sources = append(p.Sources, source)
		}
		sort.Slice(p.Sources, func(i, j int) bool { return p.Sources[i].Identity < p.Sources[j].Identity })
		decls := map[string]goprovider.ResolvedDeclaration{}
		for _, d := range rp.Declarations {
			if _, ok := decls[d.ID]; ok {
				return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.declaration_duplicate")
			}
			decls[d.ID] = d
			org, ok := evidence.Origins[key(d.Location)]
			if !ok || !matches(org, d.Location, evidence.Sources[d.Location.File]) {
				return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.provenance_missing")
			}
			v := packagedetail.Package
			export := ""
			if d.Exported {
				v = packagedetail.Public
				export = d.Name
			}
			p.Members = append(p.Members, packagedetail.Member{Identity: d.ID, Name: d.Name, ExportName: export, Visibility: v, Origin: org})
		}
		members := mp.Members
		if members == nil {
			members = mp.Functions
		}
		used := map[string]bool{}
		for _, m := range members {
			d, ok := decls[m.ID]
			if !ok || used[m.ID] || d.Name != m.Name || d.Exported != m.Exported || key(d.Location) != (Key{m.Document, m.Line, m.Column}) {
				return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.member_mismatch")
			}
			used[m.ID] = true
			for i := range p.Members {
				if p.Members[i].Identity == m.ID {
					p.Members[i].Callable = true
					p.Members[i].Parameters = append([]string(nil), m.Parameters...)
					if m.Result != "" {
						p.Members[i].Results = []string{m.Result}
					}
				}
			}
		}
		local := map[string]bool{}
		for _, im := range rp.Imports {
			org, ok := evidence.Origins[key(im.Location)]
			if !ok || !matches(org, im.Location, evidence.Sources[im.Location.File]) {
				return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.provenance_missing")
			}
			class := packagedetail.External
			if im.Local {
				class = packagedetail.Local
				local[im.ResolvedPath] = true
			}
			p.Imports = append(p.Imports, packagedetail.Import{Alias: im.Alias, Requested: im.Path, Resolved: im.ResolvedPath, Class: class, Origin: org})
		}
		want := append([]string(nil), mp.Dependencies...)
		sort.Strings(want)
		got := []string{}
		for x := range local {
			got = append(got, x)
		}
		sort.Strings(got)
		if !eq(want, got) {
			return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.dependencies_mismatch")
		}
		g.Packages = append(g.Packages, p)
	}
	if len(meta) > 0 {
		return packagedetail.Graph{}, fmt.Errorf("go_package_adapter.resolution_missing")
	}
	if err := packagedetail.Validate(g); err != nil {
		return packagedetail.Graph{}, err
	}
	return g, nil
}
func key(x goprovider.ProjectLocation) Key { return Key{x.File, x.Line, x.Column} }
func matches(o packagedetail.Origin, l goprovider.ProjectLocation, s packagedetail.Source) bool {
	return o.Path == l.File && o.ByteStart == uint64(l.ByteStart) && o.ByteEnd == uint64(l.ByteEnd) && o.StartLine == uint32(l.Line) && o.StartColumn == uint32(l.Column) && o.EndLine == uint32(l.EndLine) && o.EndColumn == uint32(l.EndColumn) && o.SourceIdentity == s.Identity && o.ContentDigest == s.ContentDigest
}
func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
