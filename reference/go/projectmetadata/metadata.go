// Package projectmetadata extracts the typed Go-project ownership view already
// proven by a validated canonical Project artifact.
package projectmetadata

import (
	"fmt"
	"sort"

	"seme.local/reference/goprovider"
	"seme.local/reference/projectinstance"
	"seme.local/reference/wire"
)

var (
	packageSchema    = mustID("0000000000000000000000000000b010")
	interfaceSchema  = mustID("0000000000000000000000000000b011")
	dependencySchema = mustID("0000000000000000000000000000b012")
	snapshotSchema   = mustID("0000000000000000000000000000e011")
)

// Extract returns package metadata without consulting source paths or a Go
// parser. Project instance validation is the authority for every shape and
// signature read below.
func Extract(source []byte) ([]goprovider.PackageMetadata, error) {
	if err := projectinstance.Validate(source); err != nil {
		return nil, err
	}
	e, err := wire.Decode(source)
	if err != nil {
		return nil, err
	}
	var root wire.ID
	for _, entity := range e.Entities {
		if entity.Schema == snapshotSchema {
			root = entity.Fields[mustID("e113")].Reference
		}
	}
	if root == (wire.ID{}) {
		return nil, fmt.Errorf("project_metadata.snapshot")
	}
	names := map[wire.ID]string{}
	for eid, entity := range e.Entities {
		if entity.Schema == packageSchema {
			names[eid] = string(entity.Fields[mustID("b100")].Bytes)
		}
	}
	out := make([]goprovider.PackageMetadata, 0, len(names))
	for pid, name := range names {
		entity := e.Entities[pid]
		item := goprovider.PackageMetadata{Name: name, Root: pid == root}
		for _, value := range entity.Fields[mustID("b103")].List {
			dependency := e.Entities[value.Reference]
			if dependency.Schema != dependencySchema {
				return nil, fmt.Errorf("project_metadata.dependency")
			}
			target := dependency.Fields[mustID("b122")].Reference
			resolved, ok := names[target]
			if !ok {
				return nil, fmt.Errorf("project_metadata.dependency_target")
			}
			item.Dependencies = append(item.Dependencies, resolved)
		}
		for _, value := range entity.Fields[mustID("b102")].List {
			iface := e.Entities[value.Reference]
			if iface.Schema != interfaceSchema {
				return nil, fmt.Errorf("project_metadata.interface")
			}
			function := iface.Fields[mustID("b111")].Reference
			f := goprovider.PackageFunctionMetadata{ID: function.String(), Name: string(iface.Fields[mustID("b110")].Bytes), Result: iface.Fields[mustID("b113")].Reference.String()}
			for _, parameter := range iface.Fields[mustID("b112")].List {
				f.Parameters = append(f.Parameters, parameter.Reference.String())
			}
			item.Functions = append(item.Functions, f)
		}
		sort.Strings(item.Dependencies)
		sort.Slice(item.Functions, func(i, j int) bool { return item.Functions[i].ID < item.Functions[j].ID })
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func mustID(short string) wire.ID {
	for len(short) < 32 {
		short = "0" + short
	}
	id, err := wire.ParseID(short)
	if err != nil {
		panic(err)
	}
	return id
}
