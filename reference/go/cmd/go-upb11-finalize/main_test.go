package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutOutput(t *testing.T) {
	for _, arguments := range [][]string{nil, {"extra"}, {"-client-revision", "2"}} {
		var stderr bytes.Buffer
		if err := run(context.Background(), arguments, &stderr); err == nil {
			t.Fatalf("accepted %#v", arguments)
		}
	}
}
