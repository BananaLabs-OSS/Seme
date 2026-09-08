package foldproof

import (
	"math"
	"testing"
)

func TestSum(t *testing.T) {
	if Sum([3]int64{-7, 0, 42}) != 35 {
		t.Fatal("ordered fold returned wrong sum")
	}
	if SumEmpty([0]int64{}) != 0 {
		t.Fatal("empty fold did not return its initial value")
	}
	if Sum([3]int64{math.MaxInt64, 1, -1}) != math.MaxInt64 {
		t.Fatal("fold lost modular i64 behavior")
	}
}
