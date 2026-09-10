package controlledeffectsruntime

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

type stressHandler struct {
	seen       map[string]bool
	classified int
	handled    int
}

func (h *stressHandler) Classify(command []byte) Classification {
	h.classified++
	if len(command) != 3 {
		return Rejected
	}
	if command[0] == 2 {
		return Rejected
	}
	key := string(command)
	if h.seen[key] {
		return Duplicate
	}
	h.seen[key] = true
	return New
}
func (h *stressHandler) Handle(command []byte, _ ClockSample, r RandomState) SemanticResult {
	h.handled++
	next := r.Value*48271 + 1
	state := make([]byte, 8)
	binary.LittleEndian.PutUint64(state, uint64(next))
	return SemanticResult{Committed: true, Effects: []bool{command[2]%2 == 0}, State: state, Random: RandomState{Value: next, Draws: r.Draws + 1}}
}

type stressPorts struct {
	clockCalls, effectCalls uint64
	failAt                  uint64
	requests                map[string]EffectRequest
}

func (p *stressPorts) Sample(Operation) ClockOutcome {
	p.clockCalls++
	return ClockOutcome{Sample: ClockSample{UnixMilliseconds: int64(1_000 + p.clockCalls), Sequence: p.clockCalls}}
}
func (p *stressPorts) Deliver(r EffectRequest) EffectOutcome {
	p.effectCalls++
	if p.effectCalls == p.failAt {
		return EffectOutcome{Error: &PortError{Identity: "injected.failure", OperationIdentity: r.OperationIdentity}}
	}
	receipt := append([]byte("receipt:"), r.IdempotencyKey...)
	p.requests[string(receipt)] = cloneRequest(r)
	return EffectOutcome{Receipt: receipt}
}
func (p *stressPorts) Evidence(receipt []byte) (EffectRequest, bool) {
	r, ok := p.requests[string(receipt)]
	return cloneRequest(r), ok
}

type stressSummary struct {
	Committed, Delivered, Duplicate, Rejected, Retries, Failures int
	Final                                                        RandomState
	State                                                        []byte
	ClockCalls, EffectCalls                                      uint64
}

func runStress(t *testing.T) stressSummary {
	t.Helper()
	p := profile()
	ports := &stressPorts{failAt: 129, requests: map[string]EffectRequest{}}
	h := &stressHandler{seen: map[string]bool{}}
	g := grants(p)
	random := RandomState{Value: 1}
	var state []byte
	var out stressSummary
	for i := 0; i < 4096; i++ {
		var command []byte
		if i < 256 {
			command = []byte{1, byte(i >> 8), byte(i)}
		} else if i%2 == 0 {
			j := i % 256
			command = []byte{1, byte(j >> 8), byte(j)}
		} else {
			command = []byte{2, byte(i >> 8), byte(i)}
		}
		r := Execute(p, g, ports, ports, h, command, random)
		if r.SemanticCommitted {
			out.Committed++
			random = r.Random
			state = bytes.Clone(r.State)
		}
		if r.PhysicalDelivered {
			out.Delivered++
		}
		if r.Duplicate {
			out.Duplicate++
		}
		if r.Rejected {
			out.Rejected++
		}
		if r.Failure != "" {
			out.Failures++
		}
		if r.Retry != nil {
			out.Retries++
			rr := RetryEffect(p, g, ports, *r.Retry)
			if !rr.PhysicalDelivered || rr.Failure != "" {
				t.Fatalf("retry %d: %#v", i, rr)
			}
			out.Delivered++
		}
	}
	if h.classified != 4096 || h.handled != 256 {
		t.Fatalf("semantic calls classify=%d handle=%d", h.classified, h.handled)
	}
	out.Final = random
	out.State = state
	out.ClockCalls = ports.clockCalls
	out.EffectCalls = ports.effectCalls
	return out
}

func TestDeterministic4096CommandHostBoundary(t *testing.T) {
	a, b := runStress(t), runStress(t)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("nondeterministic\n%#v\n%#v", a, b)
	}
	if a.Committed != 256 || a.Delivered != 256 || a.Duplicate != 1920 || a.Rejected != 1920 || a.Retries != 1 || a.Failures != 1 || a.Final.Draws != 256 || a.ClockCalls != 256 || a.EffectCalls != 257 {
		t.Fatalf("unexpected summary: %#v", a)
	}
}
