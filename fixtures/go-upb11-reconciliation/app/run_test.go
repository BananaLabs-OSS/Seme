package app

import "testing"

func TestRun(t *testing.T) {
	if Run(41) != 42 {
		t.Fatal("wrong result")
	}
}
