package boolean

import "testing"

func TestWithinAndEnabled(t *testing.T) {
	if !WithinAndEnabled(4, 5) {
		t.Fatal("in-range value rejected")
	}
	if WithinAndEnabled(6, 5) {
		t.Fatal("out-of-range value accepted")
	}
}
