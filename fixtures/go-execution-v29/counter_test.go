package mutableclosure

import "testing"

func TestMutableClosureCommitsIndependentState(t *testing.T) {
	if got := Run(10, 5, 7); got != 22 {
		t.Fatalf("positive result = %d", got)
	}
	if got := Run(-10, -5, 7); got != -8 {
		t.Fatalf("negative result = %d", got)
	}
	left, right := MakeCounter(0), MakeCounter(100)
	if left(1) != 1 || left(2) != 3 || right(-5) != 95 || left(0) != 3 {
		t.Fatal("closure instances aliased or lost committed state")
	}
}
