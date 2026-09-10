// Package projectv4report produces deterministic, source-byte-free evidence
// from an authenticated Project Contract v4 composition.
package projectv4report

import (
	"encoding/hex"
	"fmt"

	"seme.local/reference/dependencyreport"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectv3report"
	"seme.local/reference/wire"
)

type Report struct {
	ContractModule   string                  `json:"contract_module"`
	ContractRevision string                  `json:"contract_revision"`
	ArtifactRevision string                  `json:"artifact_revision"`
	Snapshot         Snapshot                `json:"dependency_graph_snapshot"`
	Project          projectv3report.Report  `json:"project"`
	Dependency       dependencyreport.Report `json:"dependency"`
}
type Snapshot struct {
	ID                         string `json:"id"`
	ProjectGraphSnapshot       string `json:"project_graph_snapshot"`
	DependencyClosure          string `json:"dependency_closure"`
	ContentRevision            string `json:"content_revision"`
	ProjectArtifactRevision    string `json:"project_artifact_revision"`
	DependencyArtifactRevision string `json:"dependency_artifact_revision"`
}

func Inspect(in projectdependencyinstance.Inputs) (Report, error) {
	if err := projectdependencyinstance.Validate(in); err != nil {
		return Report{}, fmt.Errorf("project_v4_report.validate:%w", err)
	}
	p, err := projectv3report.Inspect(in.ProjectV3)
	if err != nil {
		return Report{}, err
	}
	d, err := dependencyreport.Inspect(in.Contracts.Dependency(), in.Dependency)
	if err != nil {
		return Report{}, err
	}
	e, err := wire.Decode(in.Composed)
	if err != nil {
		return Report{}, err
	}
	x, err := one(e, id("e019"))
	if err != nil {
		return Report{}, err
	}
	q := e.Entities[x]
	pin := in.Contracts.Project().Pin()
	return Report{
		ContractModule: pin.Module.String(), ContractRevision: pin.Revision.String(), ArtifactRevision: e.Revision.String(),
		Snapshot: Snapshot{ID: x.String(), ProjectGraphSnapshot: q.Fields[id("e190")].Reference.String(), DependencyClosure: q.Fields[id("e191")].Reference.String(), ContentRevision: hex.EncodeToString(q.Fields[id("e192")].Bytes), ProjectArtifactRevision: artifactRevision(in.ProjectV3.Composed), DependencyArtifactRevision: artifactRevision(in.Dependency)},
		Project:  p, Dependency: d,
	}, nil
}
func artifactRevision(b []byte) string {
	e, err := wire.Decode(b)
	if err != nil {
		return ""
	}
	return e.Revision.String()
}
func one(e wire.Envelope, s wire.ID) (wire.ID, error) {
	var x wire.ID
	n := 0
	for id, q := range e.Entities {
		if q.Schema == s {
			x = id
			n++
		}
	}
	if n != 1 {
		return x, fmt.Errorf("project_v4_report.root:%s:%d", s, n)
	}
	return x, nil
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
