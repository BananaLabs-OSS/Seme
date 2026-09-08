package collectionquery

import "testing"

func TestLastOr(t *testing.T) {
	for _, test := range []struct {
		name     string
		values   []int64
		fallback int64
		want     int64
	}{
		{"empty", nil, -9, -9},
		{"one", []int64{42}, -9, 42},
		{"many", []int64{-7, 0, 42}, -9, 42},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := LastOr(test.values, test.fallback); got != test.want {
				t.Fatalf("LastOr(%v, %d) = %d, want %d", test.values, test.fallback, got, test.want)
			}
		})
	}
}
