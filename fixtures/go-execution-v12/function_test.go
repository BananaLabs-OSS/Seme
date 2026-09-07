package purefunction

import "testing"

func TestAllowed(t *testing.T) {
	if !Allowed(true, 4, 5) {
		t.Fatal("enabled in-range value rejected")
	}
	if Allowed(false, 4, 5) {
		t.Fatal("disabled value accepted")
	}
	if Allowed(true, 6, 5) {
		t.Fatal("enabled out-of-range value accepted")
	}
}
