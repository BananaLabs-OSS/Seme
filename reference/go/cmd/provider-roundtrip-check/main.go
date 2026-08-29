// Command provider-roundtrip-check independently verifies the Provider
// Contract v1 Go proof. It is an oracle, not provider authority.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"seme.local/reference/goprovider"
)

func main() {
	if len(os.Args) != 7 {
		fatal("usage: provider-roundtrip-check BEFORE AFTER REPORT ORIGINAL_PROJECT CHANGED_PROJECT TARGET")
	}
	before := readManifest(os.Args[1])
	after := readManifest(os.Args[2])
	report := readReport(os.Args[3])
	target := os.Args[6]
	must(before.Revision != after.Revision, "revision did not change")
	must(report.BaseRevision == before.Revision, "projection base revision mismatch")
	must(report.ResultRevision != "" && report.ResultRevision != before.Revision, "missing committed semantic revision")
	must(report.Validation == "go test ./..." && report.ValidationStatus == 0, "native validation did not pass")
	beforeDecl := byID(before.Declarations)
	afterDecl := byID(after.Declarations)
	must(len(beforeDecl) == len(afterDecl), "declaration count changed")
	for identity, old := range beforeDecl {
		now, ok := afterDecl[identity]
		must(ok, "identity %s disappeared", identity)
		must(old.Signature == now.Signature, "signature changed for %s", identity)
		must(old.Fingerprint == now.Fingerprint, "resolved semantics changed for %s", identity)
		if identity == target {
			must(old.Name == "Greeting" && now.Name == "Welcome", "target rename mismatch")
			must(old.Qualified != now.Qualified, "target qualified name did not change")
		} else {
			must(old.Name == now.Name && old.Qualified == now.Qualified && old.NativeKey == now.NativeKey, "unaffected declaration changed: %s", identity)
		}
	}
	oldTarget, ok := beforeDecl[target]
	must(ok, "target absent before import")
	newTarget := afterDecl[target]
	must(oldTarget.MatchFingerprint == newTarget.MatchFingerprint, "identity matching evidence changed")
	must(len(report.ChangedFiles) == 1 && report.ChangedFiles[0] == "greet.go", "unexpected changed files")
	original, err := os.ReadFile(filepath.Join(os.Args[4], "greet.go"))
	check(err)
	changed, err := os.ReadFile(filepath.Join(os.Args[5], "greet.go"))
	check(err)
	expected := append([]byte(nil), original...)
	occurrences := append([]goprovider.Occurrence(nil), oldTarget.Occurrences...)
	sort.Slice(occurrences, func(i, j int) bool { return occurrences[i].Start > occurrences[j].Start })
	for _, occurrence := range occurrences {
		must(string(expected[occurrence.Start:occurrence.End]) == "Greeting", "bad original occurrence")
		expected = append(expected[:occurrence.Start], append([]byte("Welcome"), expected[occurrence.End:]...)...)
	}
	must(bytes.Equal(expected, changed), "projection changed bytes outside resolved occurrences")
	check(filepath.WalkDir(os.Args[5], func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(entry.Name(), ".seme") || entry.Name() == "manifest.json" {
			return fmt.Errorf("Seme artifact leaked into native project: %s", path)
		}
		return nil
	}))
}
func readManifest(path string) goprovider.Manifest {
	manifest, err := goprovider.ReadManifest(path)
	check(err)
	return manifest
}
func readReport(path string) goprovider.ProjectionReport {
	b, err := os.ReadFile(path)
	check(err)
	var report goprovider.ProjectionReport
	check(json.Unmarshal(b, &report))
	return report
}
func byID(values []goprovider.Declaration) map[string]goprovider.Declaration {
	out := map[string]goprovider.Declaration{}
	for _, value := range values {
		out[value.ID] = value
	}
	return out
}
func must(ok bool, format string, args ...any) {
	if !ok {
		fatal(fmt.Sprintf(format, args...))
	}
}
func check(err error) {
	if err != nil {
		fatal(err.Error())
	}
}
func fatal(message string) { fmt.Fprintln(os.Stderr, "provider-roundtrip-check:", message); os.Exit(1) }
