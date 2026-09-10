package main

import (
	"context"
	"strings"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/wire"
)

func TestRejectsIncompleteArgumentsWithoutOutput(t *testing.T) {
	var out strings.Builder
	err := run(context.Background(), []string{"-bundle", "/tmp/missing"}, &out)
	if err == nil || !strings.Contains(err.Error(), "path:") || out.Len() != 0 {
		t.Fatalf("err=%v output=%q", err, out.String())
	}
}
func TestReportRejectsUnauthenticatedEmptyArtifacts(t *testing.T) {
	if _, err := makeReport(contractSetZero(), goupb07bundle.Result{}); err == nil {
		t.Fatal("empty authority reported")
	}
}
func TestDigestFieldIsExact32Bytes(t *testing.T) {
	q := wire.Entity{Fields: map[wire.ID]wire.Value{wid("e252"): {Tag: 5, Bytes: make([]byte, 32)}}}
	if x, err := digestField(q, "e252"); err != nil || len(x) != 64 {
		t.Fatalf("x=%q err=%v", x, err)
	}
	q.Fields[wid("e252")] = wire.Value{Tag: 5, Bytes: make([]byte, 31)}
	if _, err := digestField(q, "e252"); err == nil {
		t.Fatal("short digest accepted")
	}
}

// Kept behind a helper so this test documents that a zero contract set is not
// authority; makeReport must still reject before emitting a partial report.
func contractSetZero() contractcatalog.ProjectContractSetV10 {
	return contractcatalog.ProjectContractSetV10{}
}
