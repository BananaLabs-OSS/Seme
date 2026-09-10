package persistence

import (
	"errors"
	"example.test/go-uab-11/application"
	"example.test/go-uab-11/state"
	"reflect"
	"testing"
)

type memory struct {
	value                       Loaded
	trace                       []string
	denyLoad, denyCAS, conflict bool
}

func (m *memory) load() (Loaded, error) {
	m.trace = append(m.trace, "load")
	if m.denyLoad {
		return Loaded{}, errors.New("load")
	}
	return cloneLoaded(m.value), nil
}
func (m *memory) cas(operation Operation) error {
	m.trace = append(m.trace, "compare-exchange")
	if m.denyCAS {
		return errors.New("cas")
	}
	if m.conflict {
		return errors.New("conflict")
	}
	m.value = Loaded{Found: true, Version: 2, Token: operation.Token + "-next", V2: operation.Value}
	return nil
}
func execute(m *memory, grants Grants, initial state.V2, nextDigest string, key string) (Plan, error) {
	if !grants.Read || !grants.CompareExchange {
		return Plan{Error: 74}, nil
	}
	if key == "" {
		return Plan{Error: 75}, nil
	}
	loaded, err := m.load()
	if err != nil {
		return Plan{}, err
	}
	plan := BuildPlan(grants, loaded, initial, nextDigest, key)
	if !plan.Ok {
		return plan, nil
	}
	if err = m.cas(plan.Value.CompareExchange); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func TestPurePlanVectors(t *testing.T) {
	base := application.State{Name: "pilot", Values: []int64{1}, Counters: map[int64]int64{1: 2}}
	cases := []struct {
		name           string
		loaded         Loaded
		grants         Grants
		key            string
		ok             bool
		code, revision int64
		operations     int
	}{
		{"missing", Loaded{}, Grants{true, true}, "slot", true, 0, 2, 2},
		{"migrate-v1", Loaded{Found: true, Version: 1, Token: "t4", V1: state.V1{State: base}}, Grants{true, true}, "slot", true, 0, 2, 2},
		{"existing-v2", Loaded{Found: true, Version: 2, Token: "t8", V2: state.V2{State: base, Revision: 8}}, Grants{true, true}, "slot", true, 0, 9, 2},
		{"unknown-version", Loaded{Found: true, Version: 3, Token: "t2"}, Grants{true, true}, "slot", false, 71, 0, 0},
		{"malformed-v1", Loaded{Found: true, Version: 1, Token: "t2"}, Grants{true, true}, "slot", false, 60, 0, 0},
		{"malformed-v2", Loaded{Found: true, Version: 2, Token: "t2", V2: state.V2{State: base}}, Grants{true, true}, "slot", false, 61, 0, 0},
		{"revision-overflow", Loaded{Found: true, Version: 2, Token: "t2", V2: state.V2{State: base, Revision: 9223372036854775807}}, Grants{true, true}, "slot", false, 62, 0, 0},
		{"read-denied", Loaded{}, Grants{false, true}, "slot", false, 74, 0, 0},
		{"cas-denied", Loaded{}, Grants{true, false}, "slot", false, 74, 0, 0},
		{"bad-key", Loaded{}, Grants{true, true}, "", false, 75, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			before := cloneLoaded(c.loaded)
			initial := state.V2{State: base, Revision: 1}
			nextDigest := "sha256:090807"
			got := BuildPlan(c.grants, c.loaded, initial, nextDigest, c.key)
			operations := []Operation{}
			if got.Ok {
				operations = []Operation{got.Value.Load, got.Value.CompareExchange}
			}
			if got.Ok != c.ok || got.Error != c.code || len(operations) != c.operations {
				t.Fatalf("plan=%#v", got)
			}
			if !reflect.DeepEqual(c.loaded, before) {
				t.Fatal("planner mutated loaded state")
			}
			if got.Ok {
				if got.Value.Value.Revision != c.revision || operations[0].Sequence != 0 || operations[0].Kind != 0 || operations[1].Sequence != 1 || operations[1].Kind != 1 || !reflect.DeepEqual(operations[1].Token, c.loaded.Token) || !reflect.DeepEqual(operations[1].Digest, nextDigest) {
					t.Fatalf("ordered plan=%#v", got)
				}
			}
		})
	}
}

func TestNativePostLoadRejectionsRecordOnlyLoadAndNeverCommit(t *testing.T) {
	base := application.State{Name: "pilot", Counters: map[int64]int64{}}
	for _, loaded := range []Loaded{
		{Found: true, Version: 3, Token: "opaque:unknown"},
		{Found: true, Version: 1, Token: "opaque:bad-v1"},
		{Found: true, Version: 2, Token: "opaque:bad-v2", V2: state.V2{State: base}},
		{Found: true, Version: 2, Token: "opaque:overflow", V2: state.V2{State: base, Revision: 9223372036854775807}},
	} {
		m := &memory{value: cloneLoaded(loaded)}
		before := cloneLoaded(m.value)
		plan, err := execute(m, Grants{true, true}, state.V2{State: base, Revision: 1}, "sha256:next", "slot")
		if err != nil || plan.Ok || len(m.trace) != 1 || m.trace[0] != "load" || !reflect.DeepEqual(m.value, before) {
			t.Fatalf("plan=%#v err=%v trace=%v partial=%v", plan, err, m.trace, !reflect.DeepEqual(m.value, before))
		}
	}
}

func TestNativePortFailuresNeverPartiallyCommit(t *testing.T) {
	start := Loaded{Found: true, Version: 2, Token: "t7", V2: state.V2{State: application.State{Name: "pilot", Counters: map[int64]int64{}}, Revision: 2}}
	for _, c := range []struct {
		name                string
		load, cas, conflict bool
		trace               int
	}{{"load", true, false, false, 1}, {"cas", false, true, false, 2}, {"conflict", false, false, true, 2}} {
		t.Run(c.name, func(t *testing.T) {
			m := &memory{value: cloneLoaded(start), denyLoad: c.load, denyCAS: c.cas, conflict: c.conflict}
			before := cloneLoaded(m.value)
			if _, err := execute(m, Grants{true, true}, state.V2{}, "sha256:03", "slot"); err == nil {
				t.Fatal("failure accepted")
			}
			if len(m.trace) != c.trace || !reflect.DeepEqual(m.value, before) {
				t.Fatalf("trace=%v partial=%v", m.trace, !reflect.DeepEqual(m.value, before))
			}
		})
	}
}

func TestNativeAuthorizationPreflightMakesNoPortCall(t *testing.T) {
	m := &memory{}
	plan, err := execute(m, Grants{Read: false, CompareExchange: true}, state.V2{}, "sha256:03", "slot")
	if err != nil || plan.Error != 74 || len(m.trace) != 0 {
		t.Fatalf("plan=%#v err=%v trace=%v", plan, err, m.trace)
	}
}
func cloneLoaded(x Loaded) Loaded {
	x.V1.State = cloneState(x.V1.State)
	x.V2.State = cloneState(x.V2.State)
	return x
}
func cloneState(x application.State) application.State {
	x.Values = append([]int64(nil), x.Values...)
	if x.Counters != nil {
		m := map[int64]int64{}
		for k, v := range x.Counters {
			m[k] = v
		}
		x.Counters = m
	}
	return x
}
