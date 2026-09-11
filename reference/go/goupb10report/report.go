// Package goupb10report exposes deterministic source-free placement authority.
// It reports authenticated decisions; it does not itself claim runtime parity.
package goupb10report

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/goupb10bundle"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type FidelityCounts struct {
	Exact           int `json:"exact"`
	Refined         int `json:"refined"`
	Adapted         int `json:"adapted"`
	Emulated        int `json:"emulated"`
	EmbeddedRuntime int `json:"embedded_runtime"`
	NativeIsland    int `json:"native_island"`
	Impossible      int `json:"impossible"`
}

type Resolution struct {
	Requirement string `json:"requirement"`
	Construct   string `json:"construct"`
	Rule        string `json:"rule,omitempty"`
	Dependency  string `json:"dependency,omitempty"`
	Fidelity    string `json:"fidelity"`
	Properties  int    `json:"properties"`
	Diagnostics int    `json:"diagnostics"`
}

type Boundary struct {
	Provider  string `json:"provider"`
	Consumer  string `json:"consumer"`
	Interface string `json:"interface"`
	Transport string `json:"transport"`
}

type Report struct {
	ProjectContractRevision string         `json:"project_contract_revision"`
	TargetContractRevision  string         `json:"target_contract_revision"`
	ProjectArtifactRevision string         `json:"project_artifact_revision"`
	TargetArtifactRevision  string         `json:"target_artifact_revision"`
	ProjectContentRevision  string         `json:"project_content_revision"`
	TargetName              string         `json:"target_name"`
	TargetRevision          uint64         `json:"target_revision"`
	Executable              bool           `json:"executable"`
	RequirementCount        int            `json:"requirement_count"`
	BoundaryCount           int            `json:"boundary_count"`
	Fidelities              FidelityCounts `json:"fidelities"`
	Resolutions             []Resolution   `json:"resolutions"`
	Boundaries              []Boundary     `json:"boundaries"`
}

func Inspect(input goupb10bundle.Result) (Report, error) {
	if err := targetplaninstance.Validate(input.Plan); err != nil {
		return Report{}, fmt.Errorf("go_upb10_report.plan:%w", err)
	}
	if err := projectv13instance.Validate(input.Project); err != nil {
		return Report{}, fmt.Errorf("go_upb10_report.project:%w", err)
	}
	planGraph, err := wire.Decode(input.Artifacts.TargetPlan)
	if err != nil {
		return Report{}, err
	}
	projectGraph, err := wire.Decode(input.Artifacts.ProjectV13)
	if err != nil {
		return Report{}, err
	}
	plan, err := one(planGraph, id("c014"))
	if err != nil {
		return Report{}, err
	}
	target, exists := referenced(planGraph, plan, "c141", "c010")
	if !exists {
		return Report{}, fmt.Errorf("go_upb10_report.target")
	}
	snapshot, err := one(projectGraph, id("e02c"))
	if err != nil {
		return Report{}, err
	}
	executable := plan.Fields[id("c144")].Tag == 2
	report := Report{
		ProjectContractRevision: input.Project.Contracts.Project().Pin().Revision.String(),
		TargetContractRevision:  input.Project.Contracts.Target().Pin().Revision.String(),
		ProjectArtifactRevision: projectGraph.Revision.String(), TargetArtifactRevision: planGraph.Revision.String(),
		ProjectContentRevision: hex.EncodeToString(snapshot.Fields[id("e2c2")].Bytes),
		TargetName:             string(target.Fields[id("c100")].Bytes), TargetRevision: target.Fields[id("c101")].Unsigned,
		Executable: executable,
	}
	for _, value := range plan.Fields[id("c142")].List {
		resolution, okay := planGraph.Entities[value.Reference]
		if value.Tag != 6 || !okay || resolution.Schema != id("c013") {
			return Report{}, fmt.Errorf("go_upb10_report.resolution")
		}
		requirement, okay := referenced(planGraph, resolution, "c130", "c011")
		if !okay {
			return Report{}, fmt.Errorf("go_upb10_report.requirement")
		}
		construct := requirement.Fields[id("c110")].Reference
		entry := Resolution{Requirement: requirement.ID.String(), Construct: construct.String(), Fidelity: fidelityName(resolution.Fields[id("c133")].Unsigned), Properties: len(requirement.Fields[id("c112")].List), Diagnostics: len(resolution.Fields[id("c135")].List)}
		if selected := resolution.Fields[id("c132")]; selected.Tag == 6 {
			entry.Rule = selected.Reference.String()
			rule := planGraph.Entities[selected.Reference]
			if dependency := rule.Fields[id("c124")]; dependency.Tag == 6 {
				entry.Dependency = dependency.Reference.String()
			}
		}
		increment(&report.Fidelities, resolution.Fields[id("c133")].Unsigned)
		report.Resolutions = append(report.Resolutions, entry)
	}
	for _, value := range plan.Fields[id("c143")].List {
		boundary, okay := planGraph.Entities[value.Reference]
		if value.Tag != 6 || !okay || boundary.Schema != id("c015") {
			return Report{}, fmt.Errorf("go_upb10_report.boundary")
		}
		report.Boundaries = append(report.Boundaries, Boundary{
			Provider: boundary.Fields[id("c150")].Reference.String(), Consumer: boundary.Fields[id("c151")].Reference.String(),
			Interface: boundary.Fields[id("c152")].Reference.String(), Transport: boundary.Fields[id("c153")].Reference.String(),
		})
	}
	sort.Slice(report.Resolutions, func(i, j int) bool { return report.Resolutions[i].Requirement < report.Resolutions[j].Requirement })
	sort.Slice(report.Boundaries, func(i, j int) bool {
		left, right := report.Boundaries[i], report.Boundaries[j]
		return left.Provider+left.Consumer+left.Interface+left.Transport < right.Provider+right.Consumer+right.Interface+right.Transport
	})
	report.RequirementCount, report.BoundaryCount = len(report.Resolutions), len(report.Boundaries)
	if report.RequirementCount == 0 || executable == (report.Fidelities.Impossible != 0) {
		return Report{}, fmt.Errorf("go_upb10_report.executable_consistency")
	}
	return report, nil
}

func Marshal(report Report) ([]byte, error) { return json.MarshalIndent(report, "", "  ") }

func referenced(graph wire.Envelope, owner wire.Entity, field, schema string) (wire.Entity, bool) {
	value, exists := owner.Fields[id(field)]
	entity, present := graph.Entities[value.Reference]
	return entity, exists && value.Tag == 6 && present && entity.Schema == id(schema)
}

func one(graph wire.Envelope, schema wire.ID) (wire.Entity, error) {
	var result wire.Entity
	count := 0
	for _, entity := range graph.Entities {
		if entity.Schema == schema {
			result, count = entity, count+1
		}
	}
	if count != 1 {
		return result, fmt.Errorf("go_upb10_report.cardinality:%s", schema)
	}
	return result, nil
}

func increment(counts *FidelityCounts, value uint64) {
	switch targetplaninstance.Fidelity(value) {
	case targetplaninstance.Exact:
		counts.Exact++
	case targetplaninstance.Refined:
		counts.Refined++
	case targetplaninstance.Adapted:
		counts.Adapted++
	case targetplaninstance.Emulated:
		counts.Emulated++
	case targetplaninstance.EmbeddedRuntime:
		counts.EmbeddedRuntime++
	case targetplaninstance.NativeIsland:
		counts.NativeIsland++
	case targetplaninstance.Impossible:
		counts.Impossible++
	}
}

func fidelityName(value uint64) string {
	names := []string{"exact", "refined", "adapted", "emulated", "embedded-runtime", "native-island", "impossible"}
	if value >= uint64(len(names)) {
		return "invalid"
	}
	return names[value]
}

func id(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	result, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return result
}
