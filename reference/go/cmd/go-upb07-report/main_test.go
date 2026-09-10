package main

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/goupb07bundle"
	"seme.local/reference/wire"
)

func TestHostBoundaryProbeFitsMinimumContractBounds(t *testing.T) {
	if len([]byte(hostBoundaryProbeKey)) != 1 || len(hostBoundaryProbePayload) != 1 {
		t.Fatal("profile probe exceeds a valid one-byte Durable-v1 bound")
	}
	p, failure := (probeTransformer{version: 1}).Prepare(nil)
	if failure != nil || p.Version != 1 || len(p.Bytes) != 1 || p.SHA256 != sha256.Sum256(p.Bytes) || !(probeTransformer{version: 1}).Canonical(p) {
		t.Fatal("minimum-bound probe is not canonical")
	}
}

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
