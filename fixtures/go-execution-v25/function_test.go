package collectionupdate

import (
	"slices"
	"testing"
)

func TestUpdateAndAppend(t *testing.T) {
	original := []int64{-7, 0, 42}
	result := UpdateAndAppend(original, 1, 9, 100)
	if !slices.Equal(result, []int64{-7, 9, 42, 100}) {
		t.Fatalf("result = %v", result)
	}
	if !slices.Equal(original, []int64{-7, 0, 42}) {
		t.Fatalf("input was mutated: %v", original)
	}
}
