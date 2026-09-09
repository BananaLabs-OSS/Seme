package runtimemap

import "testing"

func TestRuntimeKeyedTally(t *testing.T) {
	values := []int64{-7, 9, -7, 42, 9, -7}
	if got := Tally(values, -7); got != 3 {
		t.Fatalf("repeated negative count = %d", got)
	}
	if got := Tally(values, 5); got != 0 {
		t.Fatalf("missing count = %d", got)
	}
	reversed := []int64{-7, 9, 42, -7, 9, -7}
	if Tally(reversed, -7) != Tally(values, -7) {
		t.Fatal("logical result depends on input encounter order")
	}
}
