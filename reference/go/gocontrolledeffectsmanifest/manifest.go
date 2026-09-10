// Package gocontrolledeffectsmanifest parses bounded, strict Go-facing
// Controlled Effects v1 source-selection evidence.
package gocontrolledeffectsmanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"seme.local/reference/gocontrolledeffectsadapter"
)

const Version = "seme.controlled-effects-selection/v1"
const MaxBytes = 1 << 16

type named struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}
type bounds struct {
	MaximumSteps            int64 `json:"maximum_steps"`
	FirstClockSequence      int64 `json:"first_clock_sequence"`
	ClockTerminalSentinel   int64 `json:"clock_terminal_sentinel"`
	MaximumUnixMilliseconds int64 `json:"maximum_unix_milliseconds"`
	MinimumSeed             int64 `json:"minimum_seed"`
	MaximumSeed             int64 `json:"maximum_seed"`
	MaximumDraws            int64 `json:"maximum_draws"`
	MaximumEffects          int64 `json:"maximum_effects"`
}
type document struct {
	Version string `json:"version"`
	Clock   struct {
		Identity        string `json:"identity"`
		Owner           string `json:"owner"`
		Sample          named  `json:"sample"`
		MonotonicPolicy string `json:"monotonic_policy"`
		InjectionPolicy string `json:"injection_policy"`
	} `json:"clock"`
	Random struct {
		Identity       string `json:"identity"`
		Owner          string `json:"owner"`
		State          named  `json:"state"`
		Draw           named  `json:"draw"`
		Next           named  `json:"next"`
		Algorithm      string `json:"algorithm"`
		OverflowPolicy string `json:"overflow_policy"`
	} `json:"random"`
	Effect struct {
		Identity       string `json:"identity"`
		Owner          string `json:"owner"`
		Capability     string `json:"capability"`
		Payload        string `json:"payload"`
		DeliveryPolicy string `json:"delivery_policy"`
	} `json:"effect"`
	Application struct {
		Command  named `json:"command"`
		State    named `json:"state"`
		Result   named `json:"result"`
		Dispatch named `json:"dispatch"`
		Replay   named `json:"replay"`
	} `json:"application"`
	Replay struct {
		DuplicatePolicy string `json:"duplicate_policy"`
		RejectionPolicy string `json:"rejection_policy"`
	} `json:"replay"`
	Bounds bounds `json:"bounds"`
}

func Parse(data []byte) (gocontrolledeffectsadapter.Selection, error) {
	if len(data) == 0 || len(data) > MaxBytes {
		return gocontrolledeffectsadapter.Selection{}, fmt.Errorf("controlled_effects_manifest.size")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var x document
	if err := d.Decode(&x); err != nil {
		return gocontrolledeffectsadapter.Selection{}, fmt.Errorf("controlled_effects_manifest.json:%w", err)
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return gocontrolledeffectsadapter.Selection{}, fmt.Errorf("controlled_effects_manifest.trailing")
	}
	if x.Version != Version {
		return gocontrolledeffectsadapter.Selection{}, fmt.Errorf("controlled_effects_manifest.version")
	}
	n := func(x named) gocontrolledeffectsadapter.Named {
		return gocontrolledeffectsadapter.Named{Package: x.Package, Name: x.Name}
	}
	s := gocontrolledeffectsadapter.Selection{
		ClockIdentity: x.Clock.Identity, ClockOwner: x.Clock.Owner, ClockSample: n(x.Clock.Sample), MonotonicPolicy: x.Clock.MonotonicPolicy, InjectionPolicy: x.Clock.InjectionPolicy,
		RandomIdentity: x.Random.Identity, RandomOwner: x.Random.Owner, RandomState: n(x.Random.State), Draw: n(x.Random.Draw), Next: n(x.Random.Next), Algorithm: x.Random.Algorithm, OverflowPolicy: x.Random.OverflowPolicy,
		EffectIdentity: x.Effect.Identity, EffectOwner: x.Effect.Owner, Capability: x.Effect.Capability, Payload: x.Effect.Payload, DeliveryPolicy: x.Effect.DeliveryPolicy,
		Command: n(x.Application.Command), State: n(x.Application.State), Result: n(x.Application.Result), Dispatch: n(x.Application.Dispatch), Replay: n(x.Application.Replay),
		DuplicatePolicy: x.Replay.DuplicatePolicy, RejectionPolicy: x.Replay.RejectionPolicy,
		Bounds: gocontrolledeffectsadapter.Bounds{MaximumSteps: x.Bounds.MaximumSteps, FirstClockSequence: x.Bounds.FirstClockSequence, ClockTerminalSentinel: x.Bounds.ClockTerminalSentinel, MaximumUnixMilliseconds: x.Bounds.MaximumUnixMilliseconds, MinimumSeed: x.Bounds.MinimumSeed, MaximumSeed: x.Bounds.MaximumSeed, MaximumDraws: x.Bounds.MaximumDraws, MaximumEffects: x.Bounds.MaximumEffects},
	}
	if err := gocontrolledeffectsadapter.ValidateSelection(s); err != nil {
		return gocontrolledeffectsadapter.Selection{}, fmt.Errorf("controlled_effects_manifest.selection:%w", err)
	}
	return s, nil
}
