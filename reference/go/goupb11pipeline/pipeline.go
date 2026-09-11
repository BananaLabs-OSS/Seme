// Package goupb11pipeline binds two independently rebuilt Project-v13
// authorities and their identity-bound Patch into one neutral Project-v14
// reconciliation artifact.
package goupb11pipeline

import (
	"bytes"
	"context"
	"fmt"
	"reflect"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goupb10pipeline"
	"seme.local/reference/projectv14instance"
)

type Input struct {
	Prior, Result              goupb10pipeline.Input
	Contracts                  contractcatalog.ProjectContractSetV14
	Patch                      []byte
	ClientRevision             uint64
	NativeValidationTranscript []byte
}

type Result struct {
	Prior, Updated goupb10pipeline.Result
	ProjectV14     []byte
	ProjectInput   projectv14instance.Inputs
}

// Build exposes no partially rebuilt project or reconciliation authority.
// Both complete Project-v13 graphs must validate under the exact predecessor
// contracts and placement policy before Project-v14 can bind their change.
func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() || in.ClientRevision == 0 || len(in.Patch) == 0 || len(in.NativeValidationTranscript) == 0 {
		return Result{}, fmt.Errorf("go_upb11.authority")
	}
	if !sameContracts(in.Prior.Contracts, in.Contracts.ProjectContractSetV13) ||
		!sameContracts(in.Result.Contracts, in.Contracts.ProjectContractSetV13) ||
		!reflect.DeepEqual(in.Prior.Policy, in.Result.Policy) {
		return Result{}, fmt.Errorf("go_upb11.base_or_policy")
	}
	prior, err := goupb10pipeline.Build(ctx, in.Prior)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb11.prior:%w", err)
	}
	updated, err := goupb10pipeline.Build(ctx, in.Result)
	if err != nil {
		return Result{}, fmt.Errorf("go_upb11.result:%w", err)
	}
	if bytes.Equal(prior.ProjectV13, updated.ProjectV13) {
		return Result{}, fmt.Errorf("go_upb11.unchanged")
	}
	projectInput := projectv14instance.Inputs{
		Contracts: in.Contracts, Prior: prior.ProjectInput, Result: updated.ProjectInput,
		Patch: bytes.Clone(in.Patch), ClientRevision: in.ClientRevision,
		NativeValidationTranscript: bytes.Clone(in.NativeValidationTranscript),
	}
	project, err := projectv14instance.Emit(projectInput)
	if err != nil {
		return Result{}, err
	}
	projectInput.Artifact = project
	if err = projectv14instance.Validate(projectInput); err != nil {
		return Result{}, err
	}
	return Result{Prior: prior, Updated: updated, ProjectV14: bytes.Clone(project), ProjectInput: projectInput}, nil
}

func sameContracts(a, b contractcatalog.ProjectContractSetV13) bool {
	return a.Validated() && b.Validated() &&
		a.Foundation().Digest() == b.Foundation().Digest() &&
		a.Execution().Digest() == b.Execution().Digest() &&
		a.Package().Digest() == b.Package().Digest() &&
		a.Dependency().Digest() == b.Dependency().Digest() &&
		a.Configuration().Digest() == b.Configuration().Digest() &&
		a.Resource().Digest() == b.Resource().Digest() &&
		a.DurableState().Digest() == b.DurableState().Digest() &&
		a.Presentation().Digest() == b.Presentation().Digest() &&
		a.OrderedTransport().Digest() == b.OrderedTransport().Digest() &&
		a.ControlledEffects().Digest() == b.ControlledEffects().Digest() &&
		a.Target().Digest() == b.Target().Digest() &&
		a.Project().Digest() == b.Project().Digest()
}
