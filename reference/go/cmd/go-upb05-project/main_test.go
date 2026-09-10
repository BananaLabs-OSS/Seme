package main

import (
	"context"
	"testing"
)

func TestRejectsMissingUnexpectedWithoutOutput(t *testing.T) {
	for _, a := range [][]string{nil, {"extra"}, {"-root", "/"}} {
		if err := run(context.Background(), a); err == nil {
			t.Fatalf("accepted %#v", a)
		}
	}
}
