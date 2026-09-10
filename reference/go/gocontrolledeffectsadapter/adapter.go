// Package gocontrolledeffectsadapter resolves explicit ordinary-Go source
// selections against an authenticated Project-v11 declaration graph.
// It never derives semantic meaning from a Go declaration name.
package gocontrolledeffectsadapter

import (
	"fmt"

	"seme.local/reference/controlledeffectsinstance"
	"seme.local/reference/projectv11instance"
	"seme.local/reference/wire"
)

type Named struct{ Package, Name string }

type Bounds struct {
	MaximumSteps, FirstClockSequence, ClockTerminalSentinel int64
	MaximumUnixMilliseconds, MinimumSeed, MaximumSeed       int64
	MaximumDraws, MaximumEffects                            int64
	MaximumCommandBytes, MaximumInitialStateBytes           int64
	MaximumTranscriptBytes                                  int64
}

type Selection struct {
	ClockIdentity, ClockOwner, MonotonicPolicy, InjectionPolicy      string
	ClockSample                                                      Named
	RandomIdentity, RandomOwner, Algorithm, OverflowPolicy           string
	RandomState, Draw, Next                                          Named
	EffectIdentity, EffectOwner, Capability, Payload, DeliveryPolicy string
	Command, State, Result, Dispatch, Replay                         Named
	DuplicatePolicy, RejectionPolicy                                 string
	Bounds                                                           Bounds
}

// Model contains only identities authenticated by the Project-v11 graph and
// immutable scalar policy evidence. Callers cannot mutate graph-owned data.
type Model = controlledeffectsinstance.Model

func Resolve(project projectv11instance.Inputs, selection Selection) (Model, error) {
	if err := projectv11instance.Validate(project); err != nil {
		return Model{}, fmt.Errorf("go_controlled_effects.project:%w", err)
	}
	e, err := wire.Decode(project.Composed)
	if err != nil {
		return Model{}, fmt.Errorf("go_controlled_effects.decode:%w", err)
	}
	return resolveEnvelope(e, selection)
}

func resolveEnvelope(e wire.Envelope, s Selection) (Model, error) {
	if err := ValidateSelection(s); err != nil {
		return Model{}, err
	}
	packages := map[string]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema != id("b010") {
			continue
		}
		name := string(q.Fields[id("b100")].Bytes)
		if name == "" || packages[name] != (wire.ID{}) {
			return Model{}, fmt.Errorf("go_controlled_effects.package")
		}
		packages[name] = x
	}
	owners := declarationOwners(e)
	owner := func(name string) (wire.ID, error) {
		x := packages[name]
		if x == (wire.ID{}) {
			return x, fmt.Errorf("go_controlled_effects.owner:%s", name)
		}
		return x, nil
	}
	resolve := func(n Named, schema string) (wire.ID, error) {
		o, err := owner(n.Package)
		if err != nil {
			return wire.ID{}, err
		}
		var found wire.ID
		for x, q := range e.Entities {
			if q.Schema == id(schema) && declarationName(q, schema) == n.Name && owners[x] == o {
				if found != (wire.ID{}) {
					return wire.ID{}, fmt.Errorf("go_controlled_effects.duplicate:%s:%s", n.Package, n.Name)
				}
				found = x
			}
		}
		if found == (wire.ID{}) {
			return found, fmt.Errorf("go_controlled_effects.missing:%s:%s", n.Package, n.Name)
		}
		return found, nil
	}
	var m Model
	var err error
	if m.ClockOwner, err = owner(s.ClockOwner); err != nil {
		return Model{}, err
	}
	if m.RandomOwner, err = owner(s.RandomOwner); err != nil {
		return Model{}, err
	}
	if m.EffectOwner, err = owner(s.EffectOwner); err != nil {
		return Model{}, err
	}
	for target, n := range map[*wire.ID]Named{&m.ClockSample: s.ClockSample, &m.RandomState: s.RandomState, &m.Draw: s.Draw, &m.Command: s.Command, &m.State: s.State, &m.Result: s.Result} {
		*target, err = resolve(n, "9030")
		if err != nil {
			return Model{}, err
		}
	}
	m.DispatchFunction, err = resolve(s.Dispatch, "9011")
	if err != nil {
		return Model{}, err
	}
	if !functionSignature(e, m.DispatchFunction, []wire.ID{m.State, m.Command}, m.Result) {
		return Model{}, fmt.Errorf("go_controlled_effects.dispatch_signature")
	}
	m.ReplayFunction, err = resolve(s.Replay, "9011")
	if err != nil {
		return Model{}, err
	}
	if m.ReplayFunction == m.DispatchFunction || !functionSignature(e, m.ReplayFunction, []wire.ID{m.State, m.Command}, m.Result) {
		return Model{}, fmt.Errorf("go_controlled_effects.replay_signature")
	}
	m.NextFunction, err = resolve(s.Next, "9011")
	if err != nil {
		return Model{}, err
	}
	if !functionSignature(e, m.NextFunction, []wire.ID{m.RandomState}, m.Draw) {
		return Model{}, fmt.Errorf("go_controlled_effects.next_signature")
	}
	m.Clock = controlledeffectsinstance.Clock{Identity: s.ClockIdentity, MonotonicPolicy: s.MonotonicPolicy, InjectionPolicy: s.InjectionPolicy}
	m.Random = controlledeffectsinstance.Random{Identity: s.RandomIdentity, Algorithm: s.Algorithm, OverflowPolicy: s.OverflowPolicy}
	m.ExternalEffect = controlledeffectsinstance.ExternalBooleanEffect{Identity: s.EffectIdentity, CapabilityIdentity: s.Capability, EffectIdentity: s.EffectIdentity, DeliveryPolicy: s.DeliveryPolicy}
	m.Replay = controlledeffectsinstance.Replay{InitialSeed: s.Bounds.MinimumSeed, Steps: []controlledeffectsinstance.ReplayStep{}, DuplicatePolicy: s.DuplicatePolicy, RejectionPolicy: s.RejectionPolicy}
	m.Bounds = controlledeffectsinstance.Bounds{MaximumSteps: uint64(s.Bounds.MaximumSteps), FirstClockSequence: uint64(s.Bounds.FirstClockSequence), ClockTerminalSentinel: uint64(s.Bounds.ClockTerminalSentinel), MaximumUnixMilliseconds: s.Bounds.MaximumUnixMilliseconds, MinimumSeed: s.Bounds.MinimumSeed, MaximumSeed: s.Bounds.MaximumSeed, MaximumDraws: uint64(s.Bounds.MaximumDraws), MaximumEffects: uint64(s.Bounds.MaximumEffects), MaximumCommandBytes: uint64(s.Bounds.MaximumCommandBytes), MaximumInitialStateBytes: uint64(s.Bounds.MaximumInitialStateBytes), MaximumTranscriptBytes: uint64(s.Bounds.MaximumTranscriptBytes)}
	return m, nil
}

func ValidateSelection(s Selection) error {
	if s.ClockIdentity != "clock.injected.unix-milliseconds.v1" || s.MonotonicPolicy != "nondecreasing" || s.InjectionPolicy != "explicit-replayable-input" ||
		s.RandomIdentity != "random.seeded.lcg-48271-plus-1.v1" || s.Algorithm != "state*48271+1" || s.OverflowPolicy != "signed-i64-modular" ||
		s.EffectIdentity != "observability.log" || s.Capability != "observability.log" || s.Payload != "bool" || s.DeliveryPolicy != "request-not-delivery" ||
		s.DuplicatePolicy != "cached-no-new-effects" || s.RejectionPolicy != "atomic-no-effects" {
		return fmt.Errorf("go_controlled_effects.policy")
	}
	want := Bounds{256, 1, 257, 4102444800000, 1, int64(^uint64(0) >> 1), 256, 256, 4096, 524288, 1048576}
	if s.Bounds != want {
		return fmt.Errorf("go_controlled_effects.bounds")
	}
	if s.ClockOwner == "" || s.RandomOwner == "" || s.EffectOwner == "" {
		return fmt.Errorf("go_controlled_effects.owner")
	}
	if s.ClockSample.Package != s.ClockOwner || s.RandomState.Package != s.RandomOwner || s.Draw.Package != s.RandomOwner || s.Next.Package != s.RandomOwner || s.Command.Package != s.EffectOwner || s.State.Package != s.EffectOwner || s.Result.Package != s.EffectOwner || s.Dispatch.Package != s.EffectOwner || s.Replay.Package != s.EffectOwner {
		return fmt.Errorf("go_controlled_effects.selection_owner")
	}
	items := []Named{s.ClockSample, s.RandomState, s.Draw, s.Next, s.Command, s.State, s.Result, s.Dispatch, s.Replay}
	seen := map[Named]bool{}
	for _, n := range items {
		if n.Package == "" || n.Name == "" {
			return fmt.Errorf("go_controlled_effects.selection")
		}
		if seen[n] {
			return fmt.Errorf("go_controlled_effects.selection_duplicate")
		}
		seen[n] = true
	}
	return nil
}

func functionSignature(e wire.Envelope, fn wire.ID, params []wire.ID, result wire.ID) bool {
	q, ok := e.Entities[fn]
	if !ok {
		return false
	}
	ps := q.Fields[id("9111")].List
	if len(ps) != len(params) || q.Fields[id("9112")].Reference != result {
		return false
	}
	for i, p := range ps {
		parameter, ok := e.Entities[p.Reference]
		if !ok || parameter.Schema != id("9012") || parameter.Fields[id("9121")].Reference != params[i] {
			return false
		}
	}
	return true
}

func declarationOwners(e wire.Envelope) map[wire.ID]wire.ID {
	out, details := map[wire.ID]wire.ID{}, map[wire.ID]wire.ID{}
	for x, q := range e.Entities {
		if q.Schema == id("b021") {
			o := q.Fields[id("b210")].Reference
			details[x] = o
			for _, v := range q.Fields[id("b211")].List {
				if member, ok := e.Entities[v.Reference]; ok {
					out[member.Fields[id("b220")].Reference] = o
				}
			}
		}
	}
	for _, q := range e.Entities {
		if q.Schema == id("b028") {
			out[q.Fields[id("b280")].Reference] = details[q.Fields[id("b281")].Reference]
		}
	}
	return out
}
func declarationName(q wire.Entity, schema string) string {
	if schema == "9030" {
		return string(q.Fields[id("9300")].Bytes)
	}
	return string(q.Fields[id("9110")].Bytes)
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
