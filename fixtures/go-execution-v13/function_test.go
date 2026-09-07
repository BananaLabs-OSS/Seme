package controltext

import "testing"

func TestDecide(t *testing.T) {
	tests := []struct {
		name         string
		enabled      bool
		value, limit int64
		want         bool
	}{
		{"enabled and in range", true, 4, 5, true},
		{"enabled and equal", true, 5, 5, true},
		{"enabled and out of range", true, 6, 5, false},
		{"disabled", false, 4, 5, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Decide(test.enabled, test.value, test.limit); got != test.want {
				t.Fatalf("Decide(%v, %d, %d) = %v, want %v", test.enabled, test.value, test.limit, got, test.want)
			}
		})
	}
}
