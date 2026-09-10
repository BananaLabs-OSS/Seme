package controlledeffectsinstance

import (
	"bytes"
	"testing"

	"seme.local/reference/wire"
)

func TestReplayDigestIsDeterministicAndMeaningSensitive(t *testing.T) {
	r := Replay{InitialSeed: 7, Steps: []ReplayStep{{CommandSequence: 1, UnixMilliseconds: 2, ClockSequence: 3, RandomDraw: 4, EffectValue: true}}, DuplicatePolicy: "exact", RejectionPolicy: "atomic"}
	a, b := replayDigest(r), replayDigest(r)
	if len(a) != 32 || !bytes.Equal(a, b) {
		t.Fatal("nondeterministic replay digest")
	}
	r.Steps[0].EffectValue = false
	if bytes.Equal(a, replayDigest(r)) {
		t.Fatal("digest ignored effect value")
	}
	r.Steps[0].EffectValue = true
	r.Steps[0].ClockSequence++
	if bytes.Equal(a, replayDigest(r)) {
		t.Fatal("digest ignored clock sequence")
	}
}

func TestModelRejectsOwnershipSignaturePurityAndReplayAdversaries(t *testing.T) {
	e, m := validModelFixture()
	if err := validateModel(e, m); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		edit func(*wire.Envelope, *Model)
	}{
		{"wrong-owner", func(_ *wire.Envelope, m *Model) { m.RandomOwner = id("dead") }},
		{"wrong-type", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.Draw]
			q.Schema = id("9011")
			e.Entities[m.Draw] = q
		}},
		{"wrong-next-signature", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.NextFunction]
			q.Fields[id("9112")] = ref(m.State)
			e.Entities[m.NextFunction] = q
		}},
		{"wrong-replay-signature", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.ReplayFunction]
			q.Fields[id("9112")] = ref(m.State)
			e.Entities[m.ReplayFunction] = q
		}},
		{"impure-next", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.NextFunction]
			q.Fields[id("9113")] = ref(id("effop"))
			e.Entities[m.NextFunction] = q
			e.Entities[id("effop")] = entity(id("effop"), "90f1", nil)
		}},
		{"impure-replay", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.ReplayFunction]
			q.Fields[id("9113")] = ref(id("effop"))
			e.Entities[m.ReplayFunction] = q
			e.Entities[id("effop")] = entity(id("effop"), "90f1", nil)
		}},
		{"same-dispatch-replay", func(_ *wire.Envelope, m *Model) { m.ReplayFunction = m.DispatchFunction }},
		{"reordered-command", func(_ *wire.Envelope, m *Model) { m.Replay.Steps[1].CommandSequence = 1 }},
		{"reordered-clock", func(_ *wire.Envelope, m *Model) { m.Replay.Steps[1].ClockSequence = 1 }},
		{"reversed-time", func(_ *wire.Envelope, m *Model) { m.Replay.Steps[1].UnixMilliseconds = 1 }},
		{"step-overflow", func(_ *wire.Envelope, m *Model) { m.Bounds.MaximumSteps = 1 }},
		{"seed-underflow", func(_ *wire.Envelope, m *Model) { m.Replay.InitialSeed = 0 }},
		{"clock-sentinel", func(_ *wire.Envelope, m *Model) { m.Replay.Steps[1].ClockSequence = m.Bounds.ClockTerminalSentinel }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, model := cloneFixture(e, m)
			tt.edit(&x, &model)
			if err := validateModel(x, model); err == nil {
				t.Fatal("accepted adversary")
			}
		})
	}
}

func validModelFixture() (wire.Envelope, Model) {
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{}}
	clockOwner, randomOwner, effectOwner := id("aa01"), id("aa02"), id("aa03")
	clockSample, randomState, draw := id("bb01"), id("bb02"), id("bb03")
	command, state, result := id("bb04"), id("bb05"), id("bb06")
	next, dispatch, replay := id("cc01"), id("cc02"), id("cc03")
	for _, x := range []wire.ID{clockSample, randomState, draw, command, state, result} {
		e.Entities[x] = entity(x, "9030", nil)
	}
	addFunction := func(x wire.ID, params []wire.ID, out wire.ID) {
		refs := make([]wire.Value, len(params))
		for i, typ := range params {
			p := stable(x[:], "param", string(rune(i)))
			e.Entities[p] = entity(p, "9012", map[string]wire.Value{"9121": ref(typ)})
			refs[i] = ref(p)
		}
		body := stable(x[:], "body")
		e.Entities[body] = entity(body, "9013", nil)
		e.Entities[x] = entity(x, "9011", map[string]wire.Value{"9111": {Tag: 7, List: refs}, "9112": ref(out), "9113": ref(body)})
	}
	addFunction(next, []wire.ID{randomState}, draw)
	addFunction(dispatch, []wire.ID{state, command}, result)
	addFunction(replay, []wire.ID{state, command}, result)
	byOwner := map[wire.ID][]wire.ID{clockOwner: {clockSample}, randomOwner: {randomState, draw, next}, effectOwner: {command, state, result, dispatch, replay}}
	for owner, declarations := range byOwner {
		detail := stable(owner[:], "detail")
		members := []wire.Value{}
		for _, declaration := range declarations {
			member := stable(owner[:], "member", declaration.String())
			e.Entities[member] = entity(member, "b022", map[string]wire.Value{"b220": ref(declaration)})
			members = append(members, ref(member))
			source := stable(owner[:], "source", declaration.String())
			e.Entities[source] = entity(source, "b028", map[string]wire.Value{"b280": ref(declaration), "b281": ref(detail)})
		}
		e.Entities[detail] = entity(detail, "b021", map[string]wire.Value{"b210": ref(owner), "b211": {Tag: 7, List: members}})
	}
	m := Model{
		ClockOwner: clockOwner, ClockSample: clockSample,
		RandomOwner: randomOwner, RandomState: randomState, Draw: draw, NextFunction: next,
		EffectOwner: effectOwner, Command: command, State: state, Result: result, DispatchFunction: dispatch, ReplayFunction: replay,
		Clock: Clock{"clock.injected", "nondecreasing", "explicit"}, Random: Random{"random.seeded", "lcg", "modular"},
		ExternalEffect: ExternalBooleanEffect{"log", "log-capability", "log-effect", "request"},
		Replay:         Replay{InitialSeed: 1, DuplicatePolicy: "cached", RejectionPolicy: "atomic", Steps: []ReplayStep{{1, 2, 1, 0, false}, {2, 3, 2, 1, true}}},
		Bounds:         Bounds{MaximumSteps: 2, FirstClockSequence: 1, ClockTerminalSentinel: 3, MaximumUnixMilliseconds: 3, MinimumSeed: 1, MaximumSeed: 99, MaximumDraws: 2, MaximumEffects: 1},
	}
	return e, m
}

func cloneFixture(e wire.Envelope, m Model) (wire.Envelope, Model) {
	b, _ := wire.Encode(e)
	x, _ := wire.Decode(b)
	m.Replay.Steps = append([]ReplayStep(nil), m.Replay.Steps...)
	return x, m
}

func TestZeroAuthorityAndArtifactAreRejected(t *testing.T) {
	if _, err := Emit(Inputs{}); err == nil {
		t.Fatal("accepted zero authority")
	}
	if err := Validate(Inputs{}); err == nil {
		t.Fatal("accepted zero artifact")
	}
}

func TestBooleanEncodingIsExact(t *testing.T) {
	if boolean(false).Tag != 1 || boolean(true).Tag != 2 {
		t.Fatal("boolean wire tags")
	}
}
