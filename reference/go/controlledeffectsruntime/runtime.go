// Package controlledeffectsruntime realizes the narrow host boundary selected
// by Controlled Effects v1. It owns no project or language-adapter behavior.
package controlledeffectsruntime

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"seme.local/reference/controlledeffectsinstance"
)

type Operation struct{ Identity, Capability string }
type Profile struct {
	Clock, Log                                              Operation
	MaximumSteps, FirstClockSequence, ClockTerminalSentinel uint64
	MaximumUnixMilliseconds, MaximumDraws, MaximumEffects   uint64
	authentication                                          [32]byte
}

func Authenticate(in controlledeffectsinstance.Inputs) (Profile, error) {
	if err := controlledeffectsinstance.Validate(in); err != nil {
		return Profile{}, fmt.Errorf("seme.effects.profile.instance:%w", err)
	}
	m := in.Model
	p := Profile{Clock: Operation{m.Clock.Identity, m.Clock.Identity}, Log: Operation{m.ExternalEffect.Identity, m.ExternalEffect.CapabilityIdentity}, MaximumSteps: m.Bounds.MaximumSteps, FirstClockSequence: m.Bounds.FirstClockSequence, ClockTerminalSentinel: m.Bounds.ClockTerminalSentinel, MaximumUnixMilliseconds: m.Bounds.MaximumUnixMilliseconds, MaximumDraws: m.Bounds.MaximumDraws, MaximumEffects: m.Bounds.MaximumEffects}
	if m.Clock.InjectionPolicy != "explicit-replayable-input" || m.Clock.MonotonicPolicy != "nondecreasing" || m.ExternalEffect.Identity != "observability.log" || m.ExternalEffect.EffectIdentity != "observability.log" || m.ExternalEffect.DeliveryPolicy != "request-not-delivery" || m.Bounds.MaximumEffects == 0 {
		return Profile{}, fmt.Errorf("seme.effects.profile.selection")
	}
	p.authentication = digestProfile(p)
	return p, nil
}

type Grants map[string]bool
type ClockSample struct {
	UnixMilliseconds int64
	Sequence         uint64
}
type RandomState struct {
	Value int64
	Draws uint64
}
type PortError struct{ Identity, OperationIdentity string }
type ClockOutcome struct {
	Sample ClockSample
	Error  *PortError
}
type EffectRequest struct {
	OperationIdentity string
	IdempotencyKey    []byte
	Value             bool
}
type EffectOutcome struct {
	Receipt []byte
	Error   *PortError
}
type ClockPort interface{ Sample(Operation) ClockOutcome }
type EffectPort interface {
	Deliver(EffectRequest) EffectOutcome
	Evidence([]byte) (EffectRequest, bool)
}

type Classification uint8

const (
	New Classification = iota + 1
	Duplicate
	Rejected
)

type Handler interface {
	Classify([]byte) Classification
	Handle([]byte, ClockSample, RandomState) SemanticResult
}
type SemanticResult struct {
	Committed bool
	Effects   []bool
	State     []byte
	Random    RandomState
	Failure   string
}
type Observation struct {
	Sequence  uint64
	Operation string
	Clock     *ClockOutcome
	Effect    *EffectOutcome
}
type Retry struct {
	request        EffectRequest
	authentication [32]byte
}
type Result struct {
	SemanticCommitted, PhysicalDelivered, Duplicate, Rejected bool
	State                                                     []byte
	Random                                                    RandomState
	Failure                                                   string
	Retry                                                     *Retry
	Trace                                                     []Observation
}

func Execute(p Profile, grants Grants, clock ClockPort, effects EffectPort, h Handler, command []byte, random RandomState) Result {
	if !validProfile(p) || clock == nil || effects == nil || h == nil || len(command) == 0 {
		return Result{Failure: "seme.effects.input.invalid"}
	}
	c := h.Classify(bytes.Clone(command))
	if c == Duplicate {
		return Result{Duplicate: true}
	}
	if c == Rejected {
		return Result{Rejected: true}
	}
	if c != New {
		return Result{Failure: "seme.effects.classification.invalid"}
	}
	// Every selected host operation is authorized before either port is called.
	if !grants[p.Clock.Capability] || !grants[p.Log.Capability] {
		return Result{Failure: "seme.effects.unauthorized"}
	}
	co := cloneClock(clock.Sample(p.Clock))
	trace := []Observation{{Sequence: 1, Operation: p.Clock.Identity, Clock: &co}}
	if co.Error != nil || co.Sample.Sequence < p.FirstClockSequence || co.Sample.Sequence >= p.ClockTerminalSentinel || co.Sample.UnixMilliseconds < 0 || uint64(co.Sample.UnixMilliseconds) > p.MaximumUnixMilliseconds {
		return Result{Failure: "seme.effects.clock.invalid", Trace: trace}
	}
	s := h.Handle(bytes.Clone(command), co.Sample, random)
	base := Result{SemanticCommitted: s.Committed, State: bytes.Clone(s.State), Random: s.Random, Trace: trace}
	if s.Failure != "" {
		if s.Committed || len(s.Effects) != 0 {
			base.Failure = "seme.effects.semantic.invalid"
		} else {
			base.Failure = s.Failure
		}
		return base
	}
	if !s.Committed || len(s.Effects) != 1 {
		base.Failure = "seme.effects.semantic.invalid"
		return base
	}
	req := EffectRequest{OperationIdentity: p.Log.Identity, IdempotencyKey: idempotency(command, co.Sample, random), Value: s.Effects[0]}
	eo := cloneEffect(effects.Deliver(cloneRequest(req)))
	base.Trace = append(base.Trace, Observation{Sequence: 2, Operation: p.Log.Identity, Effect: &eo})
	if !validDelivery(p, effects, req, eo) {
		base.Failure = "seme.effects.delivery.failed"
		base.Retry = retryFor(p, req)
		return base
	}
	base.PhysicalDelivered = true
	return base
}

func RetryEffect(p Profile, grants Grants, effects EffectPort, retry Retry) Result {
	if !validProfile(p) || effects == nil || retry.authentication != retryDigest(p, retry.request) {
		return Result{Failure: "seme.effects.retry.invalid"}
	}
	base := Result{SemanticCommitted: true}
	if !grants[p.Log.Capability] {
		base.Failure = "seme.effects.unauthorized"
		base.Retry = &retry
		return base
	}
	eo := cloneEffect(effects.Deliver(cloneRequest(retry.request)))
	base.Trace = []Observation{{Sequence: 2, Operation: p.Log.Identity, Effect: &eo}}
	if !validDelivery(p, effects, retry.request, eo) {
		base.Failure = "seme.effects.delivery.failed"
		base.Retry = &retry
		return base
	}
	base.PhysicalDelivered = true
	return base
}

type ReplayStep struct {
	Command      []byte
	Clock        ClockSample
	RandomBefore RandomState
}
type ReplayHandler interface {
	Fold([]byte, []byte, ClockSample, RandomState) SemanticResult
}

func Replay(p Profile, h ReplayHandler, initial []byte, steps []ReplayStep) Result {
	if !validProfile(p) || h == nil || uint64(len(steps)) > p.MaximumSteps {
		return Result{Failure: "seme.effects.replay.invalid"}
	}
	state := bytes.Clone(initial)
	var random RandomState
	for i, x := range steps {
		if x.Clock.Sequence < p.FirstClockSequence || x.Clock.Sequence >= p.ClockTerminalSentinel || x.Clock.UnixMilliseconds < 0 || uint64(x.Clock.UnixMilliseconds) > p.MaximumUnixMilliseconds || (i > 0 && x.Clock.Sequence <= steps[i-1].Clock.Sequence) {
			return Result{Failure: "seme.effects.replay.invalid"}
		}
		s := h.Fold(bytes.Clone(state), bytes.Clone(x.Command), x.Clock, x.RandomBefore)
		if !s.Committed || s.Failure != "" || len(s.Effects) != 1 {
			return Result{Failure: "seme.effects.replay.semantic"}
		}
		state = bytes.Clone(s.State)
		random = s.Random
	}
	return Result{SemanticCommitted: true, State: state, Random: random}
}

func validProfile(p Profile) bool {
	return p.Clock.Identity != "" && p.Clock.Capability != "" && p.Log.Identity == "observability.log" && p.Log.Capability != "" && p.MaximumSteps > 0 && p.FirstClockSequence < p.ClockTerminalSentinel && p.MaximumEffects > 0 && p.authentication == digestProfile(p)
}
func digestProfile(p Profile) [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("seme.effects.profile.v1\x00%s\x00%s\x00%s\x00%s\x00%d\x00%d\x00%d\x00%d\x00%d\x00%d", p.Clock.Identity, p.Clock.Capability, p.Log.Identity, p.Log.Capability, p.MaximumSteps, p.FirstClockSequence, p.ClockTerminalSentinel, p.MaximumUnixMilliseconds, p.MaximumDraws, p.MaximumEffects)))
}
func idempotency(c []byte, s ClockSample, r RandomState) []byte {
	h := sha256.New()
	h.Write([]byte("seme.effects.request.v1\x00"))
	h.Write(c)
	h.Write([]byte(fmt.Sprintf("\x00%d\x00%d\x00%d\x00%d", s.UnixMilliseconds, s.Sequence, r.Value, r.Draws)))
	return h.Sum(nil)
}
func retryFor(p Profile, r EffectRequest) *Retry {
	x := &Retry{request: cloneRequest(r)}
	x.authentication = retryDigest(p, x.request)
	return x
}
func retryDigest(p Profile, r EffectRequest) [32]byte {
	d := digestProfile(p)
	h := sha256.New()
	h.Write([]byte("seme.effects.retry.v1\x00"))
	h.Write(d[:])
	h.Write([]byte(r.OperationIdentity))
	h.Write(r.IdempotencyKey)
	if r.Value {
		h.Write([]byte{1})
	} else {
		h.Write([]byte{0})
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}
func validDelivery(p Profile, port EffectPort, r EffectRequest, o EffectOutcome) bool {
	if o.Error != nil || len(o.Receipt) == 0 {
		return false
	}
	got, ok := port.Evidence(bytes.Clone(o.Receipt))
	return ok && got.OperationIdentity == p.Log.Identity && got.Value == r.Value && bytes.Equal(got.IdempotencyKey, r.IdempotencyKey)
}
func cloneRequest(r EffectRequest) EffectRequest {
	r.IdempotencyKey = bytes.Clone(r.IdempotencyKey)
	return r
}
func cloneClock(o ClockOutcome) ClockOutcome {
	if o.Error != nil {
		x := *o.Error
		o.Error = &x
	}
	return o
}
func cloneEffect(o EffectOutcome) EffectOutcome {
	o.Receipt = bytes.Clone(o.Receipt)
	if o.Error != nil {
		x := *o.Error
		o.Error = &x
	}
	return o
}
