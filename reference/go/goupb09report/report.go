// Package goupb09report exposes deterministic source-free Controlled Effects
// and Project-v12 authority. It makes no runtime-parity claim.
package goupb09report

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/goupb08report"
	"seme.local/reference/goupb09bundle"
	"seme.local/reference/projectv12instance"
	"seme.local/reference/wire"
)

type Clock struct {
	Identity        string `json:"identity"`
	Owner           string `json:"owner"`
	SampleType      string `json:"sample_type"`
	MonotonicPolicy string `json:"monotonic_policy"`
	InjectionPolicy string `json:"injection_policy"`
}
type Random struct {
	Identity          string `json:"identity"`
	Owner             string `json:"owner"`
	StateType         string `json:"state_type"`
	DrawType          string `json:"draw_type"`
	NextFunction      string `json:"next_function"`
	NextFunctionOwner string `json:"next_function_owner"`
	Algorithm         string `json:"algorithm"`
	OverflowPolicy    string `json:"overflow_policy"`
}
type Effect struct {
	Identity       string `json:"identity"`
	Owner          string `json:"owner"`
	Capability     string `json:"capability"`
	Effect         string `json:"effect"`
	Payload        string `json:"payload"`
	DeliveryPolicy string `json:"delivery_policy"`
}
type Function struct {
	Identity string `json:"identity"`
	Owner    string `json:"owner"`
}
type Bounds struct {
	MaximumSteps             uint64 `json:"maximum_steps"`
	FirstClockSequence       uint64 `json:"first_clock_sequence"`
	ClockTerminalSentinel    uint64 `json:"clock_terminal_sentinel"`
	MaximumUnixMilliseconds  int64  `json:"maximum_unix_milliseconds"`
	MinimumSeed              int64  `json:"minimum_seed"`
	MaximumSeed              int64  `json:"maximum_seed"`
	MaximumDraws             uint64 `json:"maximum_draws"`
	MaximumEffects           uint64 `json:"maximum_effects"`
	MaximumCommandBytes      uint64 `json:"maximum_command_bytes"`
	MaximumInitialStateBytes uint64 `json:"maximum_initial_state_bytes"`
	MaximumTranscriptBytes   uint64 `json:"maximum_transcript_bytes"`
}
type Replay struct {
	InitialSeed        int64  `json:"initial_seed"`
	StepCount          int    `json:"step_count"`
	TranscriptSHA256   string `json:"transcript_sha256"`
	InitialStateSHA256 string `json:"initial_state_sha256"`
	DuplicatePolicy    string `json:"duplicate_policy"`
	RejectionPolicy    string `json:"rejection_policy"`
}
type Report struct {
	Transport                         goupb08report.Report `json:"transport"`
	ProjectContractRevision           string               `json:"project_contract_revision"`
	ControlledEffectsContractRevision string               `json:"controlled_effects_contract_revision"`
	ProjectArtifactRevision           string               `json:"project_artifact_revision"`
	ControlledEffectsArtifactRevision string               `json:"controlled_effects_artifact_revision"`
	ProjectContentRevision            string               `json:"project_content_revision"`
	ControlledEffectsContentRevision  string               `json:"controlled_effects_content_revision"`
	Clock                             Clock                `json:"clock"`
	Random                            Random               `json:"random"`
	Effect                            Effect               `json:"effect"`
	CommandType                       string               `json:"command_type"`
	StateType                         string               `json:"state_type"`
	ResultType                        string               `json:"result_type"`
	Dispatch                          Function             `json:"dispatch"`
	ReplayFunction                    Function             `json:"replay_function"`
	Bounds                            Bounds               `json:"bounds"`
	Replay                            Replay               `json:"replay"`
}

func Inspect(in goupb09bundle.Result) (Report, error) {
	if err := controlledeffectsinstance.Validate(in.Effects); err != nil {
		return Report{}, fmt.Errorf("go_upb09_report.effects:%w", err)
	}
	if err := projectv12instance.Validate(in.Project); err != nil {
		return Report{}, fmt.Errorf("go_upb09_report.project:%w", err)
	}
	base, err := goupb08report.Inspect(in.Base)
	if err != nil {
		return Report{}, err
	}
	ce, err := wire.Decode(in.Artifacts.ControlledEffects)
	if err != nil {
		return Report{}, err
	}
	p, err := wire.Decode(in.Artifacts.ProjectV12)
	if err != nil {
		return Report{}, err
	}
	plan, ok := one(ce, id("13100"))
	if !ok {
		return Report{}, fmt.Errorf("go_upb09_report.plan")
	}
	replay, ok := one(ce, id("13107"))
	if !ok {
		return Report{}, fmt.Errorf("go_upb09_report.replay")
	}
	snapshot, ok := one(p, id("e029"))
	if !ok {
		return Report{}, fmt.Errorf("go_upb09_report.snapshot")
	}
	m := in.Effects.Model
	z := m.Bounds
	r := Report{Transport: base, ProjectContractRevision: in.Project.Contracts.Project().Pin().Revision.String(), ControlledEffectsContractRevision: in.Effects.Contracts.ControlledEffects().Pin().Revision.String(), ProjectArtifactRevision: p.Revision.String(), ControlledEffectsArtifactRevision: ce.Revision.String(), ProjectContentRevision: hex.EncodeToString(snapshot.Fields[id("e292")].Bytes), ControlledEffectsContentRevision: hex.EncodeToString(plan.Fields[id("13207")].Bytes), Clock: Clock{m.Clock.Identity, m.ClockOwner.String(), m.ClockSample.String(), m.Clock.MonotonicPolicy, m.Clock.InjectionPolicy}, Random: Random{m.Random.Identity, m.RandomOwner.String(), m.RandomState.String(), m.Draw.String(), m.NextFunction.String(), m.RandomOwner.String(), m.Random.Algorithm, m.Random.OverflowPolicy}, Effect: Effect{m.ExternalEffect.Identity, m.EffectOwner.String(), m.ExternalEffect.CapabilityIdentity, m.ExternalEffect.EffectIdentity, "bool", m.ExternalEffect.DeliveryPolicy}, CommandType: m.Command.String(), StateType: m.State.String(), ResultType: m.Result.String(), Dispatch: Function{m.DispatchFunction.String(), m.EffectOwner.String()}, ReplayFunction: Function{m.ReplayFunction.String(), m.EffectOwner.String()}, Bounds: Bounds{z.MaximumSteps, z.FirstClockSequence, z.ClockTerminalSentinel, z.MaximumUnixMilliseconds, z.MinimumSeed, z.MaximumSeed, z.MaximumDraws, z.MaximumEffects, z.MaximumCommandBytes, z.MaximumInitialStateBytes, z.MaximumTranscriptBytes}, Replay: Replay{m.Replay.InitialSeed, len(m.Replay.Steps), hex.EncodeToString(replay.Fields[id("13272")].Bytes), hex.EncodeToString(replay.Fields[id("13276")].Bytes), m.Replay.DuplicatePolicy, m.Replay.RejectionPolicy}}
	return r, nil
}
func Marshal(r Report) ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
func one(e wire.Envelope, s wire.ID) (wire.Entity, bool) {
	var q wire.Entity
	n := 0
	for _, x := range e.Entities {
		if x.Schema == s {
			q = x
			n++
		}
	}
	return q, n == 1
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
