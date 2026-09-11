// Package goupb10deployment defines the authenticated, deterministic physical
// deployment records carried by the bounded Go UPB-10 proof. These records do
// not add target mechanics to Core: they bind a provider catalog and concrete
// launch artifacts to an already-valid Target-v1 execution plan.
package goupb10deployment

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

const PinnedPulpCommit = "acc66ca61fe69c5f2c4093bc55e13aeac6dcc001"

type Rule struct {
	Identity            string   `json:"identity"`
	Construct           string   `json:"construct"`
	MaximumRevision     uint64   `json:"maximum_revision"`
	PreservedProperties []string `json:"preserved_properties"`
	Fidelity            uint64   `json:"fidelity"`
	Dependency          string   `json:"dependency,omitempty"`
	Evidence            []string `json:"evidence"`
}

type Catalog struct {
	Format          string `json:"format"`
	AuthoritySHA256 string `json:"authority_sha256"`
	TargetName      string `json:"target_name"`
	TargetRevision  uint64 `json:"target_revision"`
	Rules           []Rule `json:"rules"`
}

type Artifact struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

type Island struct {
	Requirement string `json:"requirement"`
	Provider    string `json:"provider"`
	Consumer    string `json:"consumer"`
	Interface   string `json:"interface"`
	Transport   string `json:"transport"`
}

type LaunchManifest struct {
	Format        string     `json:"format"`
	CatalogSHA256 string     `json:"catalog_sha256"`
	PlanSHA256    string     `json:"plan_sha256"`
	ProjectSHA256 string     `json:"project_sha256"`
	PulpCommit    string     `json:"pulp_commit"`
	Artifacts     []Artifact `json:"artifacts"`
	NativeIslands []Island   `json:"native_islands"`
}

func EmitCatalog(authority []byte, target targetplaninstance.Target) ([]byte, error) {
	if len(authority) == 0 || target.Name == "" || target.Revision == 0 || len(target.Rules) == 0 {
		return nil, fmt.Errorf("deployment.catalog_input")
	}
	a := sha256.Sum256(authority)
	c := Catalog{Format: "seme-provider-catalog-v1", AuthoritySHA256: hex.EncodeToString(a[:]), TargetName: target.Name, TargetRevision: target.Revision}
	for _, r := range target.Rules {
		x := Rule{Identity: r.Identity, Construct: r.Construct.String(), MaximumRevision: r.MaximumRevision, Fidelity: uint64(r.Fidelity)}
		for _, p := range r.PreservedProperties {
			x.PreservedProperties = append(x.PreservedProperties, p.String())
		}
		for _, e := range r.Evidence {
			x.Evidence = append(x.Evidence, e.String())
		}
		if r.Dependency != nil {
			x.Dependency = r.Dependency.String()
		}
		c.Rules = append(c.Rules, x)
	}
	sort.Slice(c.Rules, func(i, j int) bool { return c.Rules[i].Identity < c.Rules[j].Identity })
	return marshal(c)
}

func ValidateCatalog(data, authority []byte, target targetplaninstance.Target) error {
	want, err := EmitCatalog(authority, target)
	if err != nil {
		return err
	}
	if !bytes.Equal(want, data) {
		return fmt.Errorf("deployment.catalog")
	}
	return nil
}

func EmitLaunch(catalog, plan, project []byte, pulpCommit string, artifacts map[string]Artifact, boundaries []targetplaninstance.Boundary) ([]byte, error) {
	if len(catalog) == 0 || len(plan) == 0 || len(project) == 0 || pulpCommit == "" || len(artifacts) == 0 {
		return nil, fmt.Errorf("deployment.launch_input")
	}
	hash := func(v []byte) string { x := sha256.Sum256(v); return hex.EncodeToString(x[:]) }
	m := LaunchManifest{Format: "seme-native-island-launch-v1", CatalogSHA256: hash(catalog), PlanSHA256: hash(plan), ProjectSHA256: hash(project), PulpCommit: pulpCommit}
	for name, a := range artifacts {
		if name == "" || a.Name != name || a.Kind == "" || len(a.SHA256) != 64 {
			return nil, fmt.Errorf("deployment.artifact")
		}
		if _, err := hex.DecodeString(a.SHA256); err != nil {
			return nil, fmt.Errorf("deployment.artifact")
		}
		m.Artifacts = append(m.Artifacts, a)
	}
	sort.Slice(m.Artifacts, func(i, j int) bool { return m.Artifacts[i].Name < m.Artifacts[j].Name })
	for _, b := range boundaries {
		m.NativeIslands = append(m.NativeIslands, Island{Requirement: b.Requirement, Provider: b.Provider.String(), Consumer: b.Consumer.String(), Interface: b.Interface.String(), Transport: b.Transport.String()})
	}
	sort.Slice(m.NativeIslands, func(i, j int) bool { return m.NativeIslands[i].Requirement < m.NativeIslands[j].Requirement })
	if len(m.NativeIslands) == 0 {
		return nil, fmt.Errorf("deployment.islands")
	}
	return marshal(m)
}

func ValidateLaunch(data, catalog, plan, project []byte, pulpCommit string, artifacts map[string]Artifact, boundaries []targetplaninstance.Boundary) error {
	want, err := EmitLaunch(catalog, plan, project, pulpCommit, artifacts, boundaries)
	if err != nil {
		return err
	}
	if !bytes.Equal(want, data) {
		return fmt.Errorf("deployment.launch")
	}
	return nil
}

func DigestArtifact(name, kind string, data []byte) (Artifact, error) {
	if name == "" || kind == "" || len(data) == 0 {
		return Artifact{}, fmt.Errorf("deployment.artifact_input")
	}
	h := sha256.Sum256(data)
	return Artifact{Name: name, Kind: kind, SHA256: hex.EncodeToString(h[:])}, nil
}

func marshal(value any) ([]byte, error) {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Compile-time use keeps the wire identity representation deliberately shared
// with the semantic plan rather than introducing a second identifier system.
var _ wire.ID
