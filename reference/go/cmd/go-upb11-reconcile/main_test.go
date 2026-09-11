package main

import (
	"bytes"
	"testing"
)

func TestRejectsIncompleteOptionsWithoutOutput(t *testing.T) {
	for _, arguments := range [][]string{nil, {"extra"}, {"-revision", "1"}} {
		var stderr bytes.Buffer
		if err := run(arguments, &stderr); err == nil {
			t.Fatalf("accepted %#v", arguments)
		}
	}
}
