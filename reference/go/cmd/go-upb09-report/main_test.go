package main

import (
	"bytes"
	"testing"
)

func TestRejectsIncompleteArgumentsWithoutReport(t *testing.T) {
	var out bytes.Buffer
	if e := run(t.Context(), nil, &out); e == nil || out.Len() != 0 {
		t.Fatal("published unauthenticated report")
	}
}
