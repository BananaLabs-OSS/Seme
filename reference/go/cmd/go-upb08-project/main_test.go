package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutProjection(t *testing.T) {
	to := filepath.Join(t.TempDir(), "projected")
	if err := run(t.Context(), []string{"-to", to, "-module", "example.test/x"}); err == nil {
		t.Fatal("projected unauthenticated")
	}
	if _, err := os.Stat(to); !os.IsNotExist(err) {
		t.Fatal("partial projection")
	}
}
