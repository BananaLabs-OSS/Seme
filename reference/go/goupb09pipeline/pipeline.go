// Package goupb09pipeline builds the bounded controlled-effects project layer above UPB08.
package goupb09pipeline

import (
	"bytes"
	"context"
	"fmt"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/gocontrolledeffectsadapter"
	"seme.local/reference/gocontrolledeffectsmanifest"
	"seme.local/reference/goupb08pipeline"
	"seme.local/reference/projectv12instance"
)

type Input struct {
	Base      goupb08pipeline.Input
	Contracts contractcatalog.ProjectContractSetV12
	Manifest  []byte
}

type Result struct {
	Base                   goupb08pipeline.Result
	ControlledEffects      []byte
	ProjectV12             []byte
	Selection              gocontrolledeffectsadapter.Selection
	ControlledEffectsInput controlledeffectsinstance.Inputs
	ProjectInput           projectv12instance.Inputs
}

// Build returns no partial authority: every cumulative layer, independent
// selection, effects artifact, and Project-v12 composition is validated before
// the result becomes observable.
func Build(ctx context.Context, in Input) (Result, error) {
	if !in.Contracts.Validated() {
		return Result{}, fmt.Errorf("go_upb09.contracts")
	}
	base, err := goupb08pipeline.Build(ctx, in.Base)
	if err != nil {
		return Result{}, err
	}
	selection, err := gocontrolledeffectsmanifest.Parse(in.Manifest)
	if err != nil {
		return Result{}, err
	}
	model, err := gocontrolledeffectsadapter.Resolve(base.ProjectInput, selection)
	if err != nil {
		return Result{}, err
	}
	effectsInput := controlledeffectsinstance.Inputs{Contracts: in.Contracts, ProjectV11: base.ProjectInput, Model: model}
	effects, err := controlledeffectsinstance.Emit(effectsInput)
	if err != nil {
		return Result{}, err
	}
	effectsInput.Artifact = effects
	if err = controlledeffectsinstance.Validate(effectsInput); err != nil {
		return Result{}, err
	}
	projectInput := projectv12instance.Inputs{Contracts: in.Contracts, ProjectV11: base.ProjectInput, Effects: effectsInput}
	project, err := projectv12instance.Emit(projectInput)
	if err != nil {
		return Result{}, err
	}
	projectInput.Composed = project
	if err = projectv12instance.Validate(projectInput); err != nil {
		return Result{}, err
	}
	return Result{Base: base, ControlledEffects: bytes.Clone(effects), ProjectV12: bytes.Clone(project), Selection: selection, ControlledEffectsInput: effectsInput, ProjectInput: projectInput}, nil
}
