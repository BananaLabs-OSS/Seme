// Package projectv3report produces deterministic, source-byte-free evidence
// from an authenticated Project v3 composition.
package projectv3report

import (
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/wire"
)

type Report struct {
	Root     string            `json:"root"`
	Packages []Package         `json:"packages"`
	Sources  []SourceOwnership `json:"sources"`
}
type Package struct {
	Identity, ID string
	Root         bool
	Sources      []string
	Members      []Member
	Imports      []Import
}
type Member struct {
	Declaration, Name, Export, Visibility string
	Origin                                Span
}
type Import struct {
	Alias, Requested, Resolved, Class string
	Origin                            Span
}
type SourceOwnership struct {
	ID, Path, Digest, Package, Class, Preservation string
	ByteSize                                       uint64
}
type Span struct {
	Source, Path, Digest                       string
	ByteStart, ByteEnd                         uint64
	StartLine, StartColumn, EndLine, EndColumn uint64
}

func Inspect(in projectgraphinstance.Inputs) (Report, error) {
	if err := projectgraphinstance.Validate(in); err != nil {
		return Report{}, fmt.Errorf("project_v3_report.validate:%w", err)
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return Report{}, err
	}
	names := map[wire.ID]string{}
	for x, q := range e.Entities {
		if q.Schema == id("b010") {
			names[x] = string(q.Fields[id("b100")].Bytes)
		}
	}
	snap, err := one(e, id("e011"))
	if err != nil {
		return Report{}, err
	}
	root := e.Entities[snap].Fields[id("e113")].Reference
	report := Report{Root: names[root]}
	owners := map[wire.ID]string{}
	graph, err := one(e, id("b020"))
	if err != nil {
		return Report{}, err
	}
	for _, dv := range e.Entities[graph].Fields[id("b200")].List {
		d := e.Entities[dv.Reference]
		pid := d.Fields[id("b210")].Reference
		p := Package{Identity: names[pid], ID: pid.String(), Root: pid == root}
		sourceSet := map[string]bool{}
		for _, mv := range d.Fields[id("b211")].List {
			m := e.Entities[mv.Reference]
			vis := e.Entities[m.Fields[id("b222")].Reference].Fields[id("b230")].Unsigned
			labels := []string{"package", "project", "public"}
			if vis >= uint64(len(labels)) {
				return Report{}, fmt.Errorf("project_v3_report.visibility")
			}
			span, er := origin(e, m.Fields[id("b224")].Reference)
			if er != nil {
				return Report{}, er
			}
			export := ""
			if v, ok := m.Fields[id("b223")]; ok {
				export = string(v.Bytes)
			}
			p.Members = append(p.Members, Member{Declaration: m.Fields[id("b220")].Reference.String(), Name: string(m.Fields[id("b221")].Bytes), Export: export, Visibility: labels[vis], Origin: span})
			sourceSet[span.Source] = true
		}
		for _, bv := range d.Fields[id("b212")].List {
			b := e.Entities[bv.Reference]
			class := e.Entities[b.Fields[id("b242")].Reference].Fields[id("b250")].Unsigned
			label := "local"
			resolved := ""
			if class == 0 {
				resolved = names[b.Fields[id("b243")].Reference]
			} else if class == 1 {
				label = "external"
				dep := e.Entities[b.Fields[id("b244")].Reference]
				resolved = string(dep.Fields[id("b120")].Bytes)
			} else {
				return Report{}, fmt.Errorf("project_v3_report.import_class")
			}
			span, er := origin(e, b.Fields[id("b245")].Reference)
			if er != nil {
				return Report{}, er
			}
			alias := ""
			if v, ok := b.Fields[id("b240")]; ok {
				alias = string(v.Bytes)
			}
			p.Imports = append(p.Imports, Import{Alias: alias, Requested: string(b.Fields[id("b241")].Bytes), Resolved: resolved, Class: label, Origin: span})
			sourceSet[span.Source] = true
		}
		for source := range sourceSet {
			p.Sources = append(p.Sources, source)
			sid, _ := wire.ParseID(source)
			if prior, ok := owners[sid]; ok && prior != p.Identity {
				return Report{}, fmt.Errorf("project_v3_report.source_owner")
			}
			owners[sid] = p.Identity
		}
		sort.Strings(p.Sources)
		sort.Slice(p.Members, func(i, j int) bool { return p.Members[i].Declaration < p.Members[j].Declaration })
		sort.Slice(p.Imports, func(i, j int) bool {
			a, b := p.Imports[i], p.Imports[j]
			if a.Origin.Source != b.Origin.Source {
				return a.Origin.Source < b.Origin.Source
			}
			if a.Origin.ByteStart != b.Origin.ByteStart {
				return a.Origin.ByteStart < b.Origin.ByteStart
			}
			return a.Requested < b.Requested
		})
		report.Packages = append(report.Packages, p)
	}
	sort.Slice(report.Packages, func(i, j int) bool { return report.Packages[i].Identity < report.Packages[j].Identity })
	inventory, err := one(e, id("e016"))
	if err != nil {
		return Report{}, err
	}
	classes := []string{"tracked", "ignored", "generated", "vendored", "opaque"}
	preservations := []string{"byte-exact", "semantic-projection", "normalized", "regenerated"}
	for _, uv := range e.Entities[inventory].Fields[id("e164")].List {
		sid := uv.Reference
		u := e.Entities[sid]
		class := e.Entities[u.Fields[id("e153")].Reference].Fields[id("e130")].Unsigned
		pres := e.Entities[u.Fields[id("e154")].Reference].Fields[id("e140")].Unsigned
		if class >= uint64(len(classes)) || pres >= uint64(len(preservations)) {
			return Report{}, fmt.Errorf("project_v3_report.source_enum")
		}
		report.Sources = append(report.Sources, SourceOwnership{ID: sid.String(), Path: string(u.Fields[id("e150")].Bytes), Digest: hex.EncodeToString(u.Fields[id("e151")].Bytes), ByteSize: u.Fields[id("e152")].Unsigned, Package: owners[sid], Class: classes[class], Preservation: preservations[pres]})
	}
	sort.Slice(report.Sources, func(i, j int) bool { return report.Sources[i].ID < report.Sources[j].ID })
	return report, nil
}
func origin(e wire.Envelope, x wire.ID) (Span, error) {
	q, ok := e.Entities[x]
	if !ok || q.Schema != id("b026") {
		return Span{}, fmt.Errorf("project_v3_report.origin")
	}
	return Span{Source: q.Fields[id("b260")].Reference.String(), Path: string(q.Fields[id("b261")].Bytes), Digest: hex.EncodeToString(q.Fields[id("b262")].Bytes), ByteStart: q.Fields[id("b263")].Unsigned, ByteEnd: q.Fields[id("b264")].Unsigned, StartLine: q.Fields[id("b265")].Unsigned, StartColumn: q.Fields[id("b266")].Unsigned, EndLine: q.Fields[id("b267")].Unsigned, EndColumn: q.Fields[id("b268")].Unsigned}, nil
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var out wire.ID
	n := 0
	for x, q := range e.Entities {
		if q.Schema == s {
			out = x
			n++
		}
	}
	if n != 1 {
		return wire.ID{}, fmt.Errorf("project_v3_report.root:%s:%d", s, n)
	}
	return out, nil
}
func id(x string) wire.ID {
	for len(x) < 32 {
		x = "0" + x
	}
	v, e := wire.ParseID(x)
	if e != nil {
		panic(e)
	}
	return v
}
