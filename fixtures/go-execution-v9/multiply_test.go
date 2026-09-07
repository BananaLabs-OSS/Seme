package multiply

import "testing"

func TestMultiplyNested(t *testing.T) {
	tests := []struct {
		left, right, factor int64
		want                int64
	}{
		{2, 3, 4, 36},
		{-4, 5, 2, -30},
		{9, 0, 7, 0},
		{9223372036854775807, 1, 1, -9223372036854775808},
	}
	for _, test := range tests {
		if got := MultiplyNested(test.left, test.right, test.factor); got != test.want {
			t.Fatalf("MultiplyNested(%d, %d, %d) = %d, want %d", test.left, test.right, test.factor, got, test.want)
		}
	}
}
