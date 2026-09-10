package goupb09replay

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"seme.local/reference/controlledeffectsinstance"
	"testing"
)

type executor struct {
	calls int
	fail  bool
}

type alteredExecutor struct{ edit func(*Outcome) }

func (e alteredExecutor) Execute(r Request) Outcome {
	x := (&executor{}).Execute(r)
	e.edit(&x)
	return x
}

func (e *executor) Execute(r Request) Outcome {
	e.calls++
	if e.fail {
		return Outcome{Failure: "injected"}
	}
	state := append(bytes.Clone(r.State), r.CanonicalCommand...)
	response := []byte{byte(r.CommandSequence), 1}
	events := []byte{byte(r.CommandSequence), 2}
	next := r.RandomBefore*48271 + 1
	return Outcome{Committed: true, State: state, Response: response, Events: events, RandomAfter: next, RandomValue: next, RandomDrawOrdinal: r.RandomDrawOrdinal, EffectValue: r.CommandSequence%2 == 0}
}

func fixture() (controlledeffectsinstance.Replay, [32]byte, [32]byte) {
	r := controlledeffectsinstance.Replay{InitialSeed: 1, InitialState: []byte("s"), DuplicatePolicy: "cached-no-new-effects", RejectionPolicy: "atomic-no-effects", Steps: []controlledeffectsinstance.ReplayStep{}}
	x := &executor{}
	state := bytes.Clone(r.InitialState)
	random := r.InitialSeed
	for i := uint64(1); i <= 3; i++ {
		command := []byte{byte(i)}
		o := x.Execute(Request{State: state, CanonicalCommand: command, CommandSequence: i, ClockSequence: i, RandomDrawOrdinal: i, UnixMilliseconds: int64(10 + i), RandomBefore: random})
		step := controlledeffectsinstance.ReplayStep{CommandSequence: i, ClockSequence: i, UnixMilliseconds: int64(10 + i), RandomBefore: random, RandomAfter: o.RandomAfter, RandomValue: o.RandomValue, RandomDrawOrdinal: i, EffectValue: o.EffectValue, CanonicalCommand: command, ResponseSHA256: sha256.Sum256(o.Response), EventsSHA256: sha256.Sum256(o.Events), StateBeforeSHA256: sha256.Sum256(state), StateAfterSHA256: sha256.Sum256(o.State)}
		r.Steps = append(r.Steps, step)
		state = o.State
		random = o.RandomAfter
	}
	return r, transcriptDigest(r), sha256.Sum256(r.InitialState)
}

func TestVerifyCompleteReplayAndCloneOwnership(t *testing.T) {
	r, transcript, initial := fixture()
	x := &executor{}
	got, err := verify(r, transcript, initial, x)
	if err != nil || got.Steps != 3 || x.calls != 3 || !reflect.DeepEqual(got.Effects, []bool{false, true, false}) {
		t.Fatal(got, err)
	}
	r.InitialState[0] = 'X'
	r.Steps[2].CanonicalCommand[0] = 9
	if string(got.FinalState) != "s\x01\x02\x03" {
		t.Fatal("result aliased authority")
	}
}

func TestOmissionReorderAndTamperAreAtomic(t *testing.T) {
	base, transcript, initial := fixture()
	tests := map[string]func(*controlledeffectsinstance.Replay){"omission": func(r *controlledeffectsinstance.Replay) { r.Steps = r.Steps[:2] }, "reorder": func(r *controlledeffectsinstance.Replay) { r.Steps[0], r.Steps[1] = r.Steps[1], r.Steps[0] }, "clock": func(r *controlledeffectsinstance.Replay) { r.Steps[1].UnixMilliseconds = 1 }, "random": func(r *controlledeffectsinstance.Replay) { r.Steps[1].RandomAfter++ }, "effect": func(r *controlledeffectsinstance.Replay) { r.Steps[1].EffectValue = !r.Steps[1].EffectValue }, "response": func(r *controlledeffectsinstance.Replay) { r.Steps[1].ResponseSHA256[0] ^= 1 }, "events": func(r *controlledeffectsinstance.Replay) { r.Steps[1].EventsSHA256[0] ^= 1 }, "state": func(r *controlledeffectsinstance.Replay) { r.Steps[1].StateAfterSHA256[0] ^= 1 }, "command": func(r *controlledeffectsinstance.Replay) { r.Steps[1].CanonicalCommand[0] ^= 1 }, "initial": func(r *controlledeffectsinstance.Replay) { r.InitialState[0] ^= 1 }}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			r := cloneReplay(base)
			edit(&r)
			x := &executor{}
			got, err := verify(r, transcript, initial, x)
			if err == nil || !reflect.DeepEqual(got, Result{}) {
				t.Fatalf("partial result %#v %v", got, err)
			}
		})
	}
}
func TestExecutorFailureReturnsNoAuthority(t *testing.T) {
	r, tr, in := fixture()
	got, err := verify(r, tr, in, &executor{fail: true})
	if err == nil || !reflect.DeepEqual(got, Result{}) {
		t.Fatal(got, err)
	}
}

func TestDerivedOutcomeMismatchReturnsNoAuthority(t *testing.T) {
	r, transcript, initial := fixture()
	tests := map[string]func(*Outcome){
		"random-after": func(o *Outcome) { o.RandomAfter++ },
		"random-value": func(o *Outcome) { o.RandomValue++ },
		"draw-ordinal": func(o *Outcome) { o.RandomDrawOrdinal++ },
		"effect":       func(o *Outcome) { o.EffectValue = !o.EffectValue },
		"response":     func(o *Outcome) { o.Response[0] ^= 1 },
		"events":       func(o *Outcome) { o.Events[0] ^= 1 },
		"state":        func(o *Outcome) { o.State[0] ^= 1 },
		"uncommitted":  func(o *Outcome) { o.Committed = false },
	}
	for name, edit := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := verify(r, transcript, initial, alteredExecutor{edit: edit})
			if err == nil || !reflect.DeepEqual(got, Result{}) {
				t.Fatalf("partial result %#v %v", got, err)
			}
		})
	}
}
func cloneReplay(r controlledeffectsinstance.Replay) controlledeffectsinstance.Replay {
	r.InitialState = bytes.Clone(r.InitialState)
	r.Steps = append([]controlledeffectsinstance.ReplayStep(nil), r.Steps...)
	for i := range r.Steps {
		r.Steps[i].CanonicalCommand = bytes.Clone(r.Steps[i].CanonicalCommand)
	}
	return r
}
