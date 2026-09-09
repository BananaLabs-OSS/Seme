package cumulativestate

import "testing"

func TestCumulativeStateFlow(t *testing.T) {
	initial := Accumulator{Value: -7}
	enabled := Run(initial, []int64{3, 4, 5}, true)
	if enabled.State.Value != 5 || enabled.Result != 5 {
		t.Fatalf("enabled = %#v", enabled)
	}
	disabled := Run(initial, []int64{3, 4, 5}, false)
	if disabled.State.Value != -7 || disabled.Result != -7 {
		t.Fatalf("disabled = %#v", disabled)
	}
	if initial.Value != -7 {
		t.Fatalf("input state mutated: %#v", initial)
	}
}
