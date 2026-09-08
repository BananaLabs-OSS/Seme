package sliceproof

import (
	"math"
	"testing"
)

func TestSum(t *testing.T) {
	for _, test := range []struct {
		name   string
		values []int64
		want   int64
	}{
		{"empty", nil, 0},
		{"signed", []int64{-7, 0, 42}, 35},
		{"variable", []int64{1, 2, 3, 4, 5}, 15},
		{"modular", []int64{math.MaxInt64, 1, -1}, math.MaxInt64},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Sum(test.values); got != test.want {
				t.Fatalf("Sum(%v) = %d, want %d", test.values, got, test.want)
			}
		})
	}
}
