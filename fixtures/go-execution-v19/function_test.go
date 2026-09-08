package choiceproof

import "testing"

func TestChoose(t *testing.T) {
	if Choose(-7, 42, true) != 42 || Choose(-7, 42, false) != -7 {
		t.Fatal("one-sided choice did not preserve ordered mutation")
	}
}
