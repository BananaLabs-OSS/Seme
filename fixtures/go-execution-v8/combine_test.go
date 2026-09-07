package structured

import "testing"

func TestCombine(t *testing.T) {
	if got := Combine(20, 21); got != 42 {
		t.Fatalf("Combine(20, 21) = %d, want 42", got)
	}
}
