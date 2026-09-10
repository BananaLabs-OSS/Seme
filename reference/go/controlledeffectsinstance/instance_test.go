package controlledeffectsinstance

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"seme.local/reference/wire"
)

func TestReplayDigestIsDeterministicAndMeaningSensitive(t *testing.T) {
	r := Replay{InitialSeed: 7, InitialState: make([]byte, 8), Steps: []ReplayStep{{CommandSequence: 1, UnixMilliseconds: 2, ClockSequence: 3, RandomBefore: 4, RandomAfter: 5, RandomValue: 5, RandomDrawOrdinal: 1, EffectValue: true, CanonicalCommand: []byte{1}}}, DuplicatePolicy: "exact", RejectionPolicy: "atomic"}
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
		{"pure-dispatch", func(e *wire.Envelope, m *Model) {
			q := e.Entities[m.DispatchFunction]
			q.Fields[id("9113")] = ref(id("purebody"))
			e.Entities[m.DispatchFunction] = q
			e.Entities[id("purebody")] = entity(id("purebody"), "9013", nil)
		}},
		{"wrong-effect", func(e *wire.Envelope, _ *Model) {
			q := e.Entities[id("dispatch-effect")]
			target := e.Entities[q.Fields[id("9f10")].Reference]
			target.Fields[id("150")] = blob("other")
			e.Entities[target.ID] = target
		}},
		{"wrong-capability", func(e *wire.Envelope, _ *Model) {
			q := e.Entities[id("dispatch-capability")]
			q.Fields[id("160")] = blob("other")
			e.Entities[q.ID] = q
		}},
		{"wrong-payload-count", func(e *wire.Envelope, _ *Model) {
			q := e.Entities[id("dispatch-effect")]
			q.Fields[id("9f11")] = wire.Value{Tag: 7}
			e.Entities[q.ID] = q
		}},
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
	i64, stateField := id("dd01"), id("dd02")
	e.Entities[i64] = entity(i64, "9010", map[string]wire.Value{"9100": u(64), "9101": {Tag: 2}, "9102": u(0)})
	e.Entities[stateField] = entity(stateField, "9031", map[string]wire.Value{"9310": blob("value"), "9311": ref(i64), "9312": u(0)})
	e.Entities[state] = entity(state, "9030", map[string]wire.Value{"9300": blob("State"), "9301": {Tag: 7, List: []wire.Value{ref(stateField)}}})
	e.Entities[command] = entity(command, "9030", map[string]wire.Value{"9300": blob("Command"), "9301": {Tag: 7, List: []wire.Value{ref(stateField)}}})
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
	capability, effect, invocation, argument := id("dispatch-capability"), id("dispatch-effect-definition"), id("dispatch-effect"), id("dispatch-argument")
	e.Entities[capability] = entity(capability, "16", map[string]wire.Value{"160": blob("log-capability")})
	e.Entities[effect] = entity(effect, "15", map[string]wire.Value{"150": blob("log-effect"), "151": ref(capability)})
	e.Entities[argument] = entity(argument, "90b0", map[string]wire.Value{"9b00": {Tag: 2}})
	e.Entities[invocation] = entity(invocation, "90f1", map[string]wire.Value{"9f10": ref(effect), "9f11": {Tag: 7, List: []wire.Value{ref(argument)}}})
	q := e.Entities[dispatch]
	q.Fields[id("9113")] = ref(invocation)
	e.Entities[dispatch] = q
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
	initial := make([]byte, 8)
	initialSHA := sha256.Sum256(initial)
	response1, events1, after1 := sha256.Sum256([]byte("response-1")), sha256.Sum256([]byte("events-1")), sha256.Sum256([]byte("state-1"))
	response2, events2, after2 := sha256.Sum256([]byte("response-2")), sha256.Sum256([]byte("events-2")), sha256.Sum256([]byte("state-2"))
	m := Model{
		ClockOwner: clockOwner, ClockSample: clockSample,
		RandomOwner: randomOwner, RandomState: randomState, Draw: draw, NextFunction: next,
		EffectOwner: effectOwner, Command: command, State: state, Result: result, DispatchFunction: dispatch, ReplayFunction: replay,
		Clock: Clock{"clock.injected", "nondecreasing", "explicit"}, Random: Random{"random.seeded", "lcg", "modular"},
		ExternalEffect: ExternalBooleanEffect{"log", "log-capability", "log-effect", "request"},
		Replay: Replay{InitialSeed: 1, InitialState: initial, DuplicatePolicy: "cached", RejectionPolicy: "atomic", Steps: []ReplayStep{
			{CommandSequence: 1, UnixMilliseconds: 2, ClockSequence: 1, RandomBefore: 1, RandomAfter: 48272, RandomValue: 48272, RandomDrawOrdinal: 1, CanonicalCommand: make([]byte, 8), ResponseSHA256: response1, EventsSHA256: events1, StateBeforeSHA256: initialSHA, StateAfterSHA256: after1},
			{CommandSequence: 2, UnixMilliseconds: 3, ClockSequence: 2, RandomBefore: 48272, RandomAfter: 48272*48271 + 1, RandomValue: 48272*48271 + 1, RandomDrawOrdinal: 2, EffectValue: true, CanonicalCommand: []byte{1, 0, 0, 0, 0, 0, 0, 0}, ResponseSHA256: response2, EventsSHA256: events2, StateBeforeSHA256: after1, StateAfterSHA256: after2},
		}},
		Bounds: Bounds{MaximumSteps: 2, FirstClockSequence: 1, ClockTerminalSentinel: 3, MaximumUnixMilliseconds: 3, MinimumSeed: 1, MaximumSeed: 99, MaximumDraws: 2, MaximumEffects: 2, MaximumCommandBytes: 8, MaximumInitialStateBytes: 8, MaximumTranscriptBytes: 1024},
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
