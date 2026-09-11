// Package goupb10bundle authenticates the UPB-10 placement extension above a
// complete source-free UPB-09 bundle.
package goupb10bundle

import (
	"bytes"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09bundle"
	"seme.local/reference/goupb10deployment"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/targetplaninstance"
)

type Artifacts struct {
	Base                            goupb09bundle.Artifacts
	TargetPlan, ProjectV13          []byte
	ProviderCatalog, LaunchManifest []byte
	CanonicalVM, PulpCell           []byte
}

type Input struct {
	Contracts contractcatalog.ProjectContractSetV13
	Base      goupb09bundle.Result
	Policy    goprojectplacementadapter.Policy
	Artifacts Artifacts
}

type Result struct {
	Artifacts Artifacts
	Base      goupb09bundle.Result
	Plan      targetplaninstance.Inputs
	Project   projectv13instance.Inputs
}

// Load derives every new semantic decision again from the authenticated
// predecessor. Supplied plan and project bytes are comparison evidence only.
func Load(in Input) (Result, error) {
	if !in.Contracts.Validated() || !sameBaseArtifacts(in.Base.Artifacts, in.Artifacts.Base) {
		return Result{}, fmt.Errorf("go_upb10_bundle.authentication")
	}
	planInput, err := goprojectplacementadapter.Derive(in.Base.Project, in.Contracts.Target(), in.Policy)
	if err != nil {
		return Result{}, err
	}
	planInput.Artifact = bytes.Clone(in.Artifacts.TargetPlan)
	if err = targetplaninstance.Validate(planInput); err != nil {
		return Result{}, fmt.Errorf("go_upb10_bundle.plan:%w", err)
	}
	projectInput := projectv13instance.Inputs{Contracts: in.Contracts, ProjectV12: in.Base.Project, Plan: planInput, Composed: bytes.Clone(in.Artifacts.ProjectV13)}
	if err = projectv13instance.Validate(projectInput); err != nil {
		return Result{}, fmt.Errorf("go_upb10_bundle.project:%w", err)
	}
	if err = goupb10deployment.ValidateCatalog(in.Artifacts.ProviderCatalog, planInput.Authority, planInput.Model.Target); err != nil {
		return Result{}, fmt.Errorf("go_upb10_bundle.catalog:%w", err)
	}
	vm, err := goupb10deployment.DigestArtifact("canonical-vm.wasm", "wasm", in.Artifacts.CanonicalVM)
	if err != nil {
		return Result{}, err
	}
	cell, err := goupb10deployment.DigestArtifact("pulp.cell.toml", "pulp-cell-manifest", in.Artifacts.PulpCell)
	if err != nil {
		return Result{}, err
	}
	if err = goupb10deployment.ValidateLaunch(in.Artifacts.LaunchManifest, in.Artifacts.ProviderCatalog, in.Artifacts.TargetPlan, in.Artifacts.ProjectV13, goupb10deployment.PinnedPulpCommit, map[string]goupb10deployment.Artifact{vm.Name: vm, cell.Name: cell}, planInput.Model.Boundaries); err != nil {
		return Result{}, fmt.Errorf("go_upb10_bundle.launch:%w", err)
	}
	return Result{Artifacts: cloneArtifacts(in.Artifacts), Base: in.Base, Plan: planInput, Project: projectInput}, nil
}

func sameBaseArtifacts(left, right goupb09bundle.Artifacts) bool {
	a, b := flattenBase(left), flattenBase(right)
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if !bytes.Equal(a[index], b[index]) {
			return false
		}
	}
	return true
}

func flattenBase(value goupb09bundle.Artifacts) [][]byte {
	b := value.Base.Base
	return [][]byte{
		b.Construction, b.Execution, b.ProjectBase, b.Inventory, b.PackageDetail,
		b.PackageV4, b.Dependency, b.ConfigurationV3, b.ProjectV8, b.Resource,
		b.ProjectV9, b.Durable, b.Presentation, b.ProjectV10,
		value.Base.Transport, value.Base.ProjectV11, value.ControlledEffects,
		value.ProjectV12, value.ReplayAuthority,
	}
}

func cloneArtifacts(value Artifacts) Artifacts {
	clone := func(source []byte) []byte { return bytes.Clone(source) }
	b := value.Base.Base.Base
	b.Construction, b.Execution = clone(b.Construction), clone(b.Execution)
	b.ProjectBase, b.Inventory = clone(b.ProjectBase), clone(b.Inventory)
	b.PackageDetail, b.PackageV4 = clone(b.PackageDetail), clone(b.PackageV4)
	b.Dependency, b.ConfigurationV3 = clone(b.Dependency), clone(b.ConfigurationV3)
	b.ProjectV8, b.Resource, b.ProjectV9 = clone(b.ProjectV8), clone(b.Resource), clone(b.ProjectV9)
	b.Durable, b.Presentation, b.ProjectV10 = clone(b.Durable), clone(b.Presentation), clone(b.ProjectV10)
	transport := value.Base.Base
	transport.Base, transport.Transport, transport.ProjectV11 = b, clone(transport.Transport), clone(transport.ProjectV11)
	base := value.Base
	base.Base, base.ControlledEffects, base.ProjectV12, base.ReplayAuthority = transport, clone(base.ControlledEffects), clone(base.ProjectV12), clone(base.ReplayAuthority)
	return Artifacts{Base: base, TargetPlan: clone(value.TargetPlan), ProjectV13: clone(value.ProjectV13), ProviderCatalog: clone(value.ProviderCatalog), LaunchManifest: clone(value.LaunchManifest), CanonicalVM: clone(value.CanonicalVM), PulpCell: clone(value.PulpCell)}
}
