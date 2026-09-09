package application

import "testing"

func TestApply(t *testing.T) {
	for _, test := range []struct{ input, want int64 }{{0, -1}, {4, 7}, {-3, -7}} {
		if got := Apply(test.input); got != test.want {
			t.Fatalf("Apply(%d) = %d, want %d", test.input, got, test.want)
		}
	}
}
