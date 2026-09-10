// Package dependencyreport produces deterministic, source-byte-free evidence
// from an authenticated Dependency Contract v1 instance.
package dependencyreport

import (
	"encoding/hex"
	"fmt"
	"sort"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/dependencyresolution"
	"seme.local/reference/wire"
)

type Report struct {
	ContractModule   string        `json:"contract_module"`
	ContractRevision string        `json:"contract_revision"`
	ArtifactRevision string        `json:"artifact_revision"`
	ContentRevision  string        `json:"content_revision"`
	Requirements     []Requirement `json:"requirements"`
	Resolutions      []Resolution  `json:"resolutions"`
}
type Requirement struct {
	Identity    string     `json:"identity"`
	Requirement string     `json:"requirement"`
	Kind        string     `json:"kind"`
	Metadata    []Metadata `json:"metadata"`
}
type Resolution struct {
	Identity           string     `json:"identity"`
	Ecosystem          string     `json:"ecosystem,omitempty"`
	Version            string     `json:"version"`
	Kind               string     `json:"kind"`
	Dependencies       []string   `json:"dependencies"`
	Metadata           []Metadata `json:"metadata"`
	IntegrityAlgorithm string     `json:"integrity_algorithm"`
	Integrity          string     `json:"integrity"`
	SourceKind         string     `json:"source_kind"`
	SourceIdentity     string     `json:"source_identity"`
	SourceDigest       string     `json:"source_digest"`
}
type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func Inspect(contract contractcatalog.Contract, instance []byte) (Report, error) {
	c, err := dependencyinstance.Validate(contract, instance)
	if err != nil {
		return Report{}, fmt.Errorf("dependency_report.validate:%w", err)
	}
	e, err := wire.Decode(instance)
	if err != nil {
		return Report{}, err
	}
	pin := contract.Pin()
	out := Report{ContractModule: pin.Module.String(), ContractRevision: pin.Revision.String(), ArtifactRevision: e.Revision.String()}
	for _, q := range e.Entities {
		if q.Schema == id("f010") {
			v, ok := q.Fields[id("f102")]
			if !ok || v.Tag != 5 {
				return Report{}, fmt.Errorf("dependency_report.content_revision")
			}
			out.ContentRevision = hex.EncodeToString(v.Bytes)
		}
	}
	if out.ContentRevision == "" {
		return Report{}, fmt.Errorf("dependency_report.closure")
	}
	for _, r := range c.Requirements {
		out.Requirements = append(out.Requirements, Requirement{Identity: r.Identity, Requirement: r.Requirement, Kind: kind(r.Kind), Metadata: metadata(r.Metadata)})
	}
	for _, x := range c.Entries {
		out.Resolutions = append(out.Resolutions, Resolution{Identity: x.Identity, Ecosystem: x.Ecosystem, Version: x.Version, Kind: kind(x.Kind), Dependencies: append([]string(nil), x.Dependencies...), Metadata: metadata(x.Metadata), IntegrityAlgorithm: x.IntegrityAlgorithm, Integrity: x.Integrity, SourceKind: x.SourceKind, SourceIdentity: x.Source, SourceDigest: x.Digest})
	}
	sort.Slice(out.Requirements, func(i, j int) bool { return out.Requirements[i].Identity < out.Requirements[j].Identity })
	sort.Slice(out.Resolutions, func(i, j int) bool { return out.Resolutions[i].Identity < out.Resolutions[j].Identity })
	return out, nil
}
func metadata(in []dependencyresolution.Metadata) []Metadata {
	out := make([]Metadata, 0, len(in))
	for _, x := range in {
		out = append(out, Metadata{Key: x.Key, Value: x.Value})
	}
	if out == nil {
		return []Metadata{}
	}
	return out
}
func kind(k dependencyresolution.Kind) string {
	if k == dependencyresolution.Local {
		return "local"
	}
	return "external"
}
func id(s string) wire.ID {
	for len(s) < 32 {
		s = "0" + s
	}
	x, err := wire.ParseID(s)
	if err != nil {
		panic(err)
	}
	return x
}
