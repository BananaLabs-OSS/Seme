package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutCreatingDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "projected")
	err := run(context.Background(), []string{"-to", destination})
	if err == nil || !strings.Contains(err.Error(), "flag_missing") {
		t.Fatalf("error=%v", err)
	}
	if _, err := strictDir(destination); err == nil {
		t.Fatal("invalid invocation created its destination")
	}
}

func TestInsideUsesPathBoundaries(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "project")
	if !inside(root, filepath.Join(root, "child")) || inside(root, root+"-other") {
		t.Fatal("path containment boundary")
	}
}
