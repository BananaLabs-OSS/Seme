// Package goupb10pipeline adds authenticated target resolution and Project-v13
// placement to the complete cumulative UPB-09 project.
package goupb10pipeline

import (
	"bytes"
	"context"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goprojectplacementadapter"
	"seme.local/reference/goupb09pipeline"
	"seme.local/reference/projectv13instance"
	"seme.local/reference/targetplaninstance"
	"seme.local/reference/wire"
)

type Input struct {
	Base      goupb09pipeline.Input
	Contracts contractcatalog.ProjectContractSetV13
	Policy    goprojectplacementadapter.Policy
}

type Result struct {
	Base         goupb09pipeline.Result
	TargetPlan   []byte
	ProjectV13   []byte
	PlanInput    targetplaninstance.Inputs
	ProjectInput projectv13instance.Inputs
}

// Build returns no partial placement authority. The complete predecessor,
// derived requirements, target plan, and Project-v13 binding all validate
// before any result becomes observable.
func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || !in.Base.Contracts.Validated() || !samePredecessorAuthorities(in.Contracts, in.Base.Contracts) {
		return Result{}, fmt.Errorf("go_upb10.contracts")
	}
	if in.Contracts.Target().Pin() != (contractcatalog.Pin{Module: xid("c000"), Revision: xid("c001")}) {
		return Result{}, fmt.Errorf("go_upb10.target")
	}
	base, err := goupb09pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	planInput, err := goprojectplacementadapter.Derive(base.ProjectInput, in.Contracts.Target(), in.Policy)
	if err != nil {
		return Result{}, err
	}
	plan, err := targetplaninstance.Emit(planInput)
	if err != nil {
		return Result{}, err
	}
	planInput.Artifact = plan
	if err = targetplaninstance.Validate(planInput); err != nil {
		return Result{}, err
	}
	projectInput := projectv13instance.Inputs{Contracts: in.Contracts, ProjectV12: base.ProjectInput, Plan: planInput}
	project, err := projectv13instance.Emit(projectInput)
	if err != nil {
		return Result{}, err
	}
	projectInput.Composed = project
	if err = projectv13instance.Validate(projectInput); err != nil {
		return Result{}, err
	}
	return Result{Base: base, TargetPlan: bytes.Clone(plan), ProjectV13: bytes.Clone(project), PlanInput: planInput, ProjectInput: projectInput}, nil
}

func samePredecessorAuthorities(next contractcatalog.ProjectContractSetV13, prior contractcatalog.ProjectContractSetV12) bool {
	return next.Execution().Digest() == prior.Execution().Digest() &&
		next.Package().Digest() == prior.Package().Digest() &&
		next.Dependency().Digest() == prior.Dependency().Digest() &&
		next.Foundation().Digest() == prior.Foundation().Digest() &&
		next.Configuration().Digest() == prior.Configuration().Digest() &&
		next.Resource().Digest() == prior.Resource().Digest() &&
		next.DurableState().Digest() == prior.DurableState().Digest() &&
		next.Presentation().Digest() == prior.Presentation().Digest() &&
		next.OrderedTransport().Digest() == prior.OrderedTransport().Digest() &&
		next.ControlledEffects().Digest() == prior.ControlledEffects().Digest()
}

func xid(value string) wire.ID {
	for len(value) < 32 {
		value = "0" + value
	}
	result, err := wire.ParseID(value)
	if err != nil {
		panic(err)
	}
	return result
}
