// Package projectv5report produces deterministic source-byte-free evidence
// from an authenticated complete Project v5 composition.
package projectv5report

import (
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/packagev3instance"
	"seme.local/reference/projectv4report"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/wire"
)

type Report struct {
	ContractModule   string                 `json:"contract_module"`
	ContractRevision string                 `json:"contract_revision"`
	ArtifactRevision string                 `json:"artifact_revision"`
	Pins             Pins                   `json:"pins"`
	Snapshot         Snapshot               `json:"complete_project_graph_snapshot"`
	PackageGraph     PackageGraph           `json:"complete_package_graph"`
	Project          projectv4report.Report `json:"project_v4"`
}
type Pin struct {
	Module   string `json:"module"`
	Revision string `json:"revision"`
}
type Pins struct {
	Execution  Pin `json:"execution"`
	Package    Pin `json:"package"`
	Dependency Pin `json:"dependency"`
	Project    Pin `json:"project"`
}
type Snapshot struct {
	ID                      string `json:"id"`
	DependencyGraphSnapshot string `json:"dependency_graph_snapshot"`
	CompletePackageGraph    string `json:"complete_package_graph"`
	ContentRevision         string `json:"content_revision"`
}
type PackageGraph struct {
	ID              string        `json:"id"`
	PackageGraph    string        `json:"package_graph"`
	ContentRevision string        `json:"content_revision"`
	Members         []V2Member    `json:"package_v2_members"`
	Declarations    []Declaration `json:"declarations"`
}
type V2Member struct {
	Binding           string `json:"binding"`
	Declaration       string `json:"declaration"`
	DeclarationSchema string `json:"declaration_schema"`
	Category          string `json:"category"`
	Owner             string `json:"owner"`
	Name              string `json:"name"`
	Export            string `json:"export,omitempty"`
	Visibility        string `json:"visibility"`
	Origin            Span   `json:"origin"`
}
type Declaration struct {
	Binding           string   `json:"binding"`
	Declaration       string   `json:"declaration"`
	Kind              string   `json:"kind"`
	Owner             string   `json:"owner"`
	Name              string   `json:"name"`
	Export            string   `json:"export,omitempty"`
	Visibility        string   `json:"visibility"`
	Origin            Span     `json:"origin"`
	Imports           []Import `json:"imports"`
	GenericDefinition string   `json:"generic_definition,omitempty"`
}
type Span struct {
	Source      string `json:"source"`
	Path        string `json:"path"`
	Digest      string `json:"digest"`
	ByteStart   uint64 `json:"byte_start"`
	ByteEnd     uint64 `json:"byte_end"`
	StartLine   uint64 `json:"start_line"`
	StartColumn uint64 `json:"start_column"`
	EndLine     uint64 `json:"end_line"`
	EndColumn   uint64 `json:"end_column"`
}
type Import struct {
	Binding   string `json:"binding"`
	Requested string `json:"requested"`
	Resolved  string `json:"resolved"`
}

func Inspect(in projectv5instance.Inputs) (Report, error) {
	if err := projectv5instance.Validate(in); err != nil {
		return Report{}, fmt.Errorf("project_v5_report.validate:%w", err)
	}
	v4, err := projectv4report.Inspect(in.ProjectV4)
	if err != nil {
		return Report{}, err
	}
	e, _ := wire.Decode(in.Composed)
	sid := one(e, id("e020"))
	s := e.Entities[sid]
	packageReport, err := InspectPackage(in.Contracts, in.PackageV2, in.PackageV3)
	if err != nil {
		return Report{}, err
	}
	report := Report{ContractModule: in.Contracts.Project().Pin().Module.String(), ContractRevision: in.Contracts.Project().Pin().Revision.String(), ArtifactRevision: e.Revision.String(), Pins: Pins{Execution: reportPin(in.Contracts.Execution()), Package: reportPin(in.Contracts.Package()), Dependency: reportPin(in.Contracts.Dependency()), Project: reportPin(in.Contracts.Project())}, Snapshot: Snapshot{ID: sid.String(), DependencyGraphSnapshot: s.Fields[id("e200")].Reference.String(), CompletePackageGraph: s.Fields[id("e201")].Reference.String(), ContentRevision: hex.EncodeToString(s.Fields[id("e202")].Bytes)}, PackageGraph: packageReport, Project: v4}
	return report, nil
}

func InspectPackage(contracts contractcatalog.ProjectContractSetV5, packageV2, packageV3 []byte) (PackageGraph, error) {
	if err := packagev3instance.Validate(contracts, packageV2, packageV3); err != nil {
		return PackageGraph{}, fmt.Errorf("project_v5_report.package_validate:%w", err)
	}
	p3, _ := wire.Decode(packageV3)
	gid := one(p3, id("b029"))
	g := p3.Entities[gid]
	names := map[wire.ID]string{}
	details := map[wire.ID]string{}
	for x, q := range p3.Entities {
		if q.Schema == id("b010") {
			names[x] = string(q.Fields[id("b100")].Bytes)
		}
	}
	for x, q := range p3.Entities {
		if q.Schema == id("b021") {
			details[x] = names[q.Fields[id("b210")].Reference]
		}
	}
	report := PackageGraph{ID: gid.String(), PackageGraph: g.Fields[id("b290")].Reference.String(), ContentRevision: hex.EncodeToString(g.Fields[id("b292")].Bytes)}
	kinds := []string{"data-type", "behavioral-interface", "receiver-callable", "generic-definition", "generic-realization"}
	visibility := []string{"package", "project", "public"}
	for detailID, owner := range details {
		detail := p3.Entities[detailID]
		for _, mv := range detail.Fields[id("b211")].List {
			member := p3.Entities[mv.Reference]
			declarationID := member.Fields[id("b220")].Reference
			declaration := p3.Entities[declarationID]
			vis := p3.Entities[member.Fields[id("b222")].Reference].Fields[id("b230")].Unsigned
			if vis >= uint64(len(visibility)) {
				return PackageGraph{}, fmt.Errorf("project_v5_report.member_visibility")
			}
			category := "nonfunction"
			if declaration.Schema == id("9011") {
				category = "function"
			}
			m := V2Member{Binding: mv.Reference.String(), Declaration: declarationID.String(), DeclarationSchema: declaration.Schema.String(), Category: category, Owner: owner, Name: string(member.Fields[id("b221")].Bytes), Visibility: visibility[vis], Origin: span(p3, p3.Entities[member.Fields[id("b224")].Reference])}
			if x, ok := member.Fields[id("b223")]; ok {
				m.Export = string(x.Bytes)
			}
			report.Members = append(report.Members, m)
		}
	}
	sort.Slice(report.Members, func(i, j int) bool {
		if report.Members[i].Owner != report.Members[j].Owner {
			return report.Members[i].Owner < report.Members[j].Owner
		}
		return report.Members[i].Declaration < report.Members[j].Declaration
	})
	for _, v := range g.Fields[id("b291")].List {
		q := p3.Entities[v.Reference]
		kind := p3.Entities[q.Fields[id("b282")].Reference].Fields[id("b270")].Unsigned
		vis := p3.Entities[q.Fields[id("b284")].Reference].Fields[id("b230")].Unsigned
		if kind >= uint64(len(kinds)) || vis >= uint64(len(visibility)) {
			return PackageGraph{}, fmt.Errorf("project_v5_report.enum")
		}
		origin := span(p3, p3.Entities[q.Fields[id("b286")].Reference])
		d := Declaration{Binding: v.Reference.String(), Declaration: q.Fields[id("b280")].Reference.String(), Kind: kinds[kind], Owner: details[q.Fields[id("b281")].Reference], Name: string(q.Fields[id("b283")].Bytes), Visibility: visibility[vis], Origin: origin}
		if x, ok := q.Fields[id("b285")]; ok {
			d.Export = string(x.Bytes)
		}
		if x, ok := q.Fields[id("b288")]; ok {
			d.GenericDefinition = x.Reference.String()
		}
		for _, iv := range q.Fields[id("b287")].List {
			b := p3.Entities[iv.Reference]
			resolved := ""
			if x, ok := b.Fields[id("b243")]; ok {
				resolved = names[x.Reference]
			} else if x, ok := b.Fields[id("b244")]; ok {
				resolved = x.Reference.String()
			}
			d.Imports = append(d.Imports, Import{Binding: iv.Reference.String(), Requested: string(b.Fields[id("b241")].Bytes), Resolved: resolved})
		}
		sort.Slice(d.Imports, func(i, j int) bool { return d.Imports[i].Binding < d.Imports[j].Binding })
		report.Declarations = append(report.Declarations, d)
	}
	sort.Slice(report.Declarations, func(i, j int) bool {
		return report.Declarations[i].Declaration < report.Declarations[j].Declaration
	})
	return report, nil
}
func reportPin(c contractcatalog.Contract) Pin {
	return Pin{Module: c.Pin().Module.String(), Revision: c.Pin().Revision.String()}
}
func span(e wire.Envelope, q wire.Entity) Span {
	return Span{Source: q.Fields[id("b260")].Reference.String(), Path: string(q.Fields[id("b261")].Bytes), Digest: hex.EncodeToString(q.Fields[id("b262")].Bytes), ByteStart: q.Fields[id("b263")].Unsigned, ByteEnd: q.Fields[id("b264")].Unsigned, StartLine: q.Fields[id("b265")].Unsigned, StartColumn: q.Fields[id("b266")].Unsigned, EndLine: q.Fields[id("b267")].Unsigned, EndColumn: q.Fields[id("b268")].Unsigned}
}
func one(e wire.Envelope, s wire.ID) wire.ID {
	var out wire.ID
	for x, q := range e.Entities {
		if q.Schema == s {
			if out != (wire.ID{}) {
				return wire.ID{}
			}
			out = x
		}
	}
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
