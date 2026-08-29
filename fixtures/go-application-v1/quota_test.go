package quota

import (
	"math"
	"testing"
)

func TestAdmit(t *testing.T) {
	cases := []struct {
		current, delta, limit int64
		want                  bool
	}{
		{40, 2, 50, true},
		{40, 20, 50, false},
		{-10, 3, -5, true},
		{math.MaxInt64, 1, 0, true},
		{0, 0, 0, true},
	}
	for _, test := range cases {
		if got := Admit(test.current, test.delta, test.limit); got != test.want {
			t.Fatalf("Admit(%d, %d, %d) = %v, want %v", test.current, test.delta, test.limit, got, test.want)
		}
	}
}
