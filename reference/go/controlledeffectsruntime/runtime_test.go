package controlledeffectsruntime

import (
	"bytes"
	"testing"
)

func profile() Profile {
	p := Profile{Clock: Operation{"clock.injected.unix-milliseconds.v1", "clock.injected.unix-milliseconds.v1"}, Log: Operation{"observability.log", "observability.log"}, RandomIdentity: "random.seeded.lcg-48271-plus-1.v1", RandomAlgorithm: "state*48271+1", RandomOverflowPolicy: "signed-i64-modular", MaximumSteps: 256, FirstClockSequence: 1, ClockTerminalSentinel: 257, MaximumUnixMilliseconds: 4102444800000, MaximumDraws: 256, MaximumEffects: 256}
	p.authentication = digestProfile(p)
	return p
}

type ports struct {
	clock, effect int
	sample        ClockOutcome
	outcome       EffectOutcome
	request       EffectRequest
}

func (p *ports) Sample(Operation) ClockOutcome { p.clock++; return p.sample }
func (p *ports) Deliver(r EffectRequest) EffectOutcome {
	p.effect++
	p.request = cloneRequest(r)
	return p.outcome
}
func (p *ports) Evidence([]byte) (EffectRequest, bool) { return cloneRequest(p.request), true }

type handler struct {
	class  Classification
	result SemanticResult
	calls  int
}

func (h *handler) Classify([]byte) Classification { return h.class }
func (h *handler) Handle(_ []byte, _ ClockSample, random RandomState) SemanticResult {
	h.calls++
	r := h.result
	r.Random = RandomState{Value: random.Value*48271 + 1, Draws: random.Draws + 1}
	return r
}

type replay struct{ calls int }

func (r *replay) Fold(s, c []byte, _ ClockSample, random RandomState) SemanticResult {
	r.calls++
	return SemanticResult{Committed: true, Effects: []bool{true}, State: append(bytes.Clone(s), c...), Random: RandomState{Value: random.Value*48271 + 1, Draws: random.Draws + 1}}
}
func good() (*ports, *handler) {
	return &ports{sample: ClockOutcome{Sample: ClockSample{10, 1}}, outcome: EffectOutcome{Receipt: []byte("ok")}}, &handler{class: New, result: SemanticResult{Committed: true, Effects: []bool{true}, State: []byte("state"), Random: RandomState{48272, 1}}}
}
func grants(p Profile) Grants { return Grants{p.Clock.Capability: true, p.Log.Capability: true} }

func TestLiveOrderedAndImmutable(t *testing.T) {
	p := profile()
	ports, h := good()
	r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{1, 0})
	if !r.SemanticCommitted || !r.PhysicalDelivered || ports.clock != 1 || ports.effect != 1 || h.calls != 1 || len(r.Trace) != 2 || r.Trace[0].Operation != p.Clock.Identity || r.Trace[1].Operation != p.Log.Identity {
		t.Fatalf("bad result: %#v", r)
	}
	ports.outcome.Receipt[0] = 'X'
	if string(r.Trace[1].Effect.Receipt) != "ok" {
		t.Fatal("trace aliases port response")
	}
}

func TestMalformedDeliveryAndUncommittedDeliverNone(t *testing.T) {
	p := profile()
	ports, h := good()
	ports.outcome = EffectOutcome{}
	r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{})
	if r.Failure != "seme.effects.delivery.failed" || r.Retry == nil {
		t.Fatal(r)
	}
	ports, h = good()
	h.result = SemanticResult{Committed: false, Effects: nil, Failure: "domain.reject"}
	r = Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{})
	if r.Failure != "domain.reject" || ports.effect != 0 {
		t.Fatal(r)
	}
}
func TestDenialNilAndStrictProfile(t *testing.T) {
	p := profile()
	ports, h := good()
	for _, g := range []Grants{{}, {p.Clock.Capability: true}} {
		r := Execute(p, g, ports, ports, h, []byte("c"), RandomState{})
		if r.Failure != "seme.effects.unauthorized" || ports.clock != 0 || ports.effect != 0 {
			t.Fatal(r)
		}
	}
	if Execute(Profile{}, grants(p), ports, ports, h, []byte("c"), RandomState{}).Failure == "" {
		t.Fatal("zero profile")
	}
	q := p
	q.MaximumSteps++
	if Execute(q, grants(p), ports, ports, h, []byte("c"), RandomState{}).Failure == "" {
		t.Fatal("tampered profile")
	}
	if Execute(p, grants(p), nil, ports, h, []byte("c"), RandomState{}).Failure == "" {
		t.Fatal("nil")
	}
}
func TestDuplicateAndRejectedHaveNoCalls(t *testing.T) {
	p := profile()
	for _, c := range []Classification{Duplicate, Rejected} {
		ports, h := good()
		h.class = c
		r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{})
		if ports.clock+ports.effect+h.calls != 0 || (!r.Duplicate && !r.Rejected) {
			t.Fatalf("%d %#v", c, r)
		}
	}
}
func TestMalformedClockAndSemanticEffects(t *testing.T) {
	p := profile()
	for _, sample := range []ClockOutcome{{Sample: ClockSample{-1, 1}}, {Sample: ClockSample{1, 257}}, {Sample: ClockSample{1, 1}, Error: &PortError{Identity: "x"}}} {
		ports, h := good()
		ports.sample = sample
		r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{})
		if r.Failure != "seme.effects.clock.invalid" || ports.effect != 0 || h.calls != 0 {
			t.Fatal(r)
		}
	}
	for _, effects := range [][]bool{nil, {true, false}} {
		ports, h := good()
		h.result.Effects = effects
		r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{})
		if r.Failure != "seme.effects.semantic.invalid" || ports.effect != 0 {
			t.Fatal(r)
		}
	}
}

func TestExhaustedRandomRejectsBeforePorts(t *testing.T) {
	p := profile()
	ports, h := good()
	r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{Value: 1, Draws: p.MaximumDraws})
	if r.Failure != "seme.effects.random.exhausted" || ports.clock+ports.effect+h.calls != 0 {
		t.Fatalf("%#v", r)
	}
}
func TestDeliveryFailureRetryIsExactAndIdempotent(t *testing.T) {
	p := profile()
	ports, h := good()
	ports.outcome = EffectOutcome{Error: &PortError{Identity: "down", OperationIdentity: p.Log.Identity}}
	r := Execute(p, grants(p), ports, ports, h, []byte("c"), RandomState{1, 0})
	if !r.SemanticCommitted || r.PhysicalDelivered || r.Retry == nil {
		t.Fatal(r)
	}
	first := bytes.Clone(ports.request.IdempotencyKey)
	ports.outcome = EffectOutcome{Receipt: []byte("ok")}
	rr := RetryEffect(p, grants(p), ports, *r.Retry)
	if !rr.PhysicalDelivered || ports.clock != 1 || h.calls != 1 || !bytes.Equal(first, ports.request.IdempotencyKey) {
		t.Fatalf("retry %#v", rr)
	}
}
func TestReplayHasNoPortsAndFoldsExplicitInputs(t *testing.T) {
	p := profile()
	ports, _ := good()
	h := &replay{}
	first := RandomState{1, 0}
	second := RandomState{first.Value*48271 + 1, 1}
	r := Replay(p, h, []byte("a"), []ReplayStep{{[]byte("b"), ClockSample{1, 1}, first}, {[]byte("c"), ClockSample{2, 2}, second}})
	if r.Failure != "" || string(r.State) != "abc" || h.calls != 2 || ports.clock+ports.effect != 0 || len(r.Trace) != 0 {
		t.Fatalf("%#v", r)
	}
}
