package subtract

import "testing"

func TestSubtractNested(t *testing.T) {
	tests := []struct {
		left, right, factor int64
		want                int64
	}{
		{20, 3, 4, 9},
		{3, 20, 4, -25},
		{-4, -9, 2, 1},
		{-9223372036854775808, 1, 0, 9223372036854775807},
	}
	for _, test := range tests {
		if got := SubtractNested(test.left, test.right, test.factor); got != test.want {
			t.Fatalf("SubtractNested(%d, %d, %d) = %d, want %d", test.left, test.right, test.factor, got, test.want)
		}
	}
}
