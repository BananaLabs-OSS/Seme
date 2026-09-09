package cumulativetext

import "testing"

func TestCumulativeTextCollectionFlow(t *testing.T) {
	if got := Describe("sum", []int64{3, 4, 5}); got != "sum:positive" {
		t.Fatalf("positive = %q", got)
	}
	if got := Describe("sum", []int64{-3, 1}); got != "sum:non-positive" {
		t.Fatalf("negative = %q", got)
	}
	if got := Describe("empty", nil); got != "empty:non-positive" {
		t.Fatalf("empty = %q", got)
	}
}
