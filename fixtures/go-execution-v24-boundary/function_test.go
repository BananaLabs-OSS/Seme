package dynamicindex

import "testing"

func TestAt(t *testing.T) {
	if got := At([]int64{-7, 0, 42}, 2); got != 42 {
		t.Fatalf("At = %d, want 42", got)
	}
}
