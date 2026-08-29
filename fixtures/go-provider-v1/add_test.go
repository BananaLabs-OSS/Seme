package greeting

import (
	"math"
	"testing"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		a, b int64
		want int64
	}{
		{20, 22, 42},
		{-7, 3, -4},
		{math.MaxInt64, 1, math.MinInt64},
		{math.MinInt64, -1, math.MaxInt64},
	}
	for _, test := range cases {
		if got := Add(test.a, test.b); got != test.want {
			t.Fatalf("Add(%d, %d) = %d, want %d", test.a, test.b, got, test.want)
		}
	}
}
