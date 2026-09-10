package main

import (
	"context"
	"strings"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutOutput(t *testing.T) {
	var output strings.Builder
	err := run(context.Background(), []string{"-bundle", "/tmp/missing"}, &output)
	if err == nil || !strings.Contains(err.Error(), "path:") {
		t.Fatalf("error=%v", err)
	}
	if output.String() != "" {
		t.Fatalf("partial report %q", output.String())
	}
}
