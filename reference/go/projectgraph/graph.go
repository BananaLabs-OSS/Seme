// Package projectgraph exposes a deterministic, validated read model of the
// bounded Package v1 graph embedded in a canonical Project artifact.
package projectgraph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/packageinstance"
	"seme.local/reference/projectinstance"
	"seme.local/reference/wire"
)

type Report struct {
	Profile          string    `json:"profile"`
	ArtifactRevision string    `json:"artifact_revision"`
	Packages         []Package `json:"packages"`
}
type Package struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Root         bool         `json:"root"`
	Revision     string       `json:"revision"`
	Dependencies []Dependency `json:"dependencies"`
	Exports      []Export     `json:"exports"`
}
type Dependency struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Package string `json:"package"`
}
type Export struct {
	Name       string   `json:"name"`
	Function   string   `json:"function"`
	Parameters []string `json:"parameters"`
	Result     string   `json:"result"`
}

var (
	packageSchema    = id("b010")
	interfaceSchema  = id("b011")
	dependencySchema = id("b012")
	snapshotSchema   = id("e011")
)

// Inspect rejects anything outside the validated Project/Package contracts
// before reporting graph facts. It does not infer source-language behavior.
func Inspect(source []byte) (Report, error) {
	if err := projectinstance.Validate(source); err != nil {
		return Report{}, err
	}
	if err := packageinstance.Validate(source); err != nil {
		return Report{}, err
	}
	envelope, err := wire.Decode(source)
	if err != nil {
		return Report{}, err
	}
	root := wire.ID{}
	for _, entity := range envelope.Entities {
		if entity.Schema == snapshotSchema {
			root = entity.Fields[id("e113")].Reference
		}
	}
	names := map[wire.ID]string{}
	for eid, entity := range envelope.Entities {
		if entity.Schema == packageSchema {
			names[eid] = string(entity.Fields[id("b100")].Bytes)
		}
	}
	report := Report{Profile: "seme.project.package-graph.v1", ArtifactRevision: envelope.Revision.String(), Packages: []Package{}}
	for pid, name := range names {
		entity := envelope.Entities[pid]
		item := Package{ID: pid.String(), Name: name, Root: pid == root, Revision: fmt.Sprintf("%x", entity.Fields[id("b101")].Bytes), Dependencies: []Dependency{}, Exports: []Export{}}
		for _, value := range entity.Fields[id("b103")].List {
			dependency := envelope.Entities[value.Reference]
			if dependency.Schema != dependencySchema {
				return Report{}, fmt.Errorf("project_graph.dependency_schema")
			}
			target := dependency.Fields[id("b122")].Reference
			targetName, ok := names[target]
			if !ok {
				return Report{}, fmt.Errorf("project_graph.dependency_target")
			}
			item.Dependencies = append(item.Dependencies, Dependency{Name: string(dependency.Fields[id("b120")].Bytes), Kind: string(dependency.Fields[id("b121")].Bytes), Package: targetName})
		}
		for _, value := range entity.Fields[id("b102")].List {
			iface := envelope.Entities[value.Reference]
			if iface.Schema != interfaceSchema {
				return Report{}, fmt.Errorf("project_graph.interface_schema")
			}
			exported := Export{Name: string(iface.Fields[id("b110")].Bytes), Function: iface.Fields[id("b111")].Reference.String(), Parameters: []string{}, Result: iface.Fields[id("b113")].Reference.String()}
			for _, parameter := range iface.Fields[id("b112")].List {
				exported.Parameters = append(exported.Parameters, parameter.Reference.String())
			}
			item.Exports = append(item.Exports, exported)
		}
		sort.Slice(item.Dependencies, func(i, j int) bool {
			a, b := item.Dependencies[i], item.Dependencies[j]
			return a.Name < b.Name || a.Name == b.Name && (a.Package < b.Package || a.Package == b.Package && a.Kind < b.Kind)
		})
		sort.Slice(item.Exports, func(i, j int) bool {
			a, b := item.Exports[i], item.Exports[j]
			return a.Name < b.Name || a.Name == b.Name && a.Function < b.Function
		})
		report.Packages = append(report.Packages, item)
	}
	sort.Slice(report.Packages, func(i, j int) bool { return report.Packages[i].Name < report.Packages[j].Name })
	return report, nil
}

func Encode(report Report) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func InspectJSON(source []byte) ([]byte, error) {
	report, err := Inspect(source)
	if err != nil {
		return nil, err
	}
	return Encode(report)
}

func id(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	value, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return value
}
