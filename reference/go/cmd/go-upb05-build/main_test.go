package main

import (
	"bytes"
	"context"
	"testing"
)

func TestRejectsMissingAndUnexpectedArgumentsWithoutArtifacts(t *testing.T) {
	for _, args := range [][]string{nil, {"extra"}, {"-revision", "1"}} {
		var stderr bytes.Buffer
		if err := run(context.Background(), args, &stderr); err == nil {
			t.Fatalf("accepted %#v", args)
		}
	}
}
