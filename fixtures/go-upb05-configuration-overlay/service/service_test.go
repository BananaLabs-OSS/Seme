package service

import (
	"reflect"
	"testing"

	"example.test/go-uab-11/application"
	"example.test/go-uab-11/configuration"
	"example.test/go-uab-11/policy"
)

func TestConfigurationDefaultsValidationAndInitializationOrder(t *testing.T) {
	state := application.State{Name: "world", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	before := clone(state)
	defaults := configuration.Input{NamePrefix: "世界", UseDefaultLimit: true}
	runtime := Initialize(defaults, state)
	if !runtime.Ok || !runtime.Value.Ready || runtime.Value.Stage != 3 || runtime.Value.Settings.Limit != 64 || runtime.Value.Policy.Stage != 2 || !reflect.DeepEqual(state, before) {
		t.Fatalf("runtime=%#v mutated=%v", runtime, !reflect.DeepEqual(state, before))
	}
	for _, test := range []struct {
		input configuration.Input
		error int64
	}{
		{configuration.Input{UseDefaultLimit: true}, 10},
		{configuration.Input{NamePrefix: "x", Limit: 0}, 11},
		{configuration.Input{NamePrefix: "x", Limit: 4097}, 12},
	} {
		got := Initialize(test.input, state)
		if got.Ok || got.Error != test.error || !reflect.DeepEqual(state, before) {
			t.Fatalf("input=%#v got=%#v mutated=%v", test.input, got, !reflect.DeepEqual(state, before))
		}
	}
	for _, limit := range []int64{1, 4096} {
		got := Initialize(configuration.Input{NamePrefix: "x", Limit: limit}, state)
		if !got.Ok || got.Value.Settings.Limit != limit {
			t.Fatalf("limit=%d got=%#v", limit, got)
		}
	}
	outOfOrder := policy.Initialize(configuration.Initialized{Settings: configuration.Settings{NamePrefix: "x", Limit: 4}, Stage: 0})
	if outOfOrder.Ok || outOfOrder.Error != 20 {
		t.Fatalf("out-of-order=%#v", outOfOrder)
	}
	for _, test := range []struct {
		configured configuration.Initialized
		prepared   policy.Initialized
		error      int64
	}{
		{configuration.Initialized{Settings: configuration.Settings{NamePrefix: "x", Limit: 4}, Stage: 0}, policy.Initialized{Limit: 4, Stage: 2}, 21},
		{configuration.Initialized{Settings: configuration.Settings{NamePrefix: "x", Limit: 4}, Stage: 1}, policy.Initialized{Limit: 4, Stage: 1}, 21},
		{configuration.Initialized{Settings: configuration.Settings{NamePrefix: "x", Limit: 4}, Stage: 1}, policy.Initialized{Limit: 5, Stage: 2}, 22},
	} {
		got := Assemble(test.configured, test.prepared, state)
		if got.Ok || got.Error != test.error || !reflect.DeepEqual(state, before) {
			t.Fatalf("assemble=%#v mutated=%v", got, !reflect.DeepEqual(state, before))
		}
	}
}

func TestLifecycleGuardsAndConfiguredApply(t *testing.T) {
	state := application.State{Name: "world", Values: []int64{2, 3}, Counters: map[int64]int64{7: 10}}
	command := application.Command{Key: 7, Index: 1, Delta: 4, Amount: 1, Scale: true}
	if got := Apply(Runtime{State: state}, command); got.Ok || got.Error != 30 {
		t.Fatalf("not-ready=%#v", got)
	}
	limited := Runtime{Ready: true, Stage: 3, Policy: policy.Initialized{Limit: 1, Stage: 2}, State: state}
	if got := Apply(limited, command); got.Ok || got.Error != 31 {
		t.Fatalf("limited=%#v", got)
	}
	got := ApplyConfigured(configuration.Input{NamePrefix: "x", UseDefaultLimit: true}, state, command)
	if !got.Ok || got.Value.Result != 31 || !reflect.DeepEqual(got.Value.State.Values, []int64{2, 21, 21}) {
		t.Fatalf("configured=%#v", got)
	}
}

func clone(state application.State) application.State {
	values := append([]int64(nil), state.Values...)
	counters := map[int64]int64{}
	for key, value := range state.Counters {
		counters[key] = value
	}
	return application.State{Name: state.Name, Values: values, Counters: counters}
}
