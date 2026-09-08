package arrayboundary

import "testing"

func TestPick(t *testing.T) {
	values := [3]int64{-7, 0, 42}
	if Pick(values, 0) != -7 || Pick(values, 2) != 42 {
		t.Fatal("array parameter returned the wrong element")
	}
}
