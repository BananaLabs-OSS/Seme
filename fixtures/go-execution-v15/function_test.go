package localproof

import "testing"

func TestScale(t *testing.T) {
	for _, test := range []struct{ left, right, want int64 }{
		{4, 5, 14}, {-3, 8, 13}, {0, 0, 0},
	} {
		if got := Scale(test.left, test.right); got != test.want {
			t.Fatalf("Scale(%d, %d) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}
