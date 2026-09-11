// Command project-roundtrip-publish atomically combines provider-projected
// semantic sources with byte-exact units from a verified source snapshot.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"seme.local/reference/projectbundle"
	"seme.local/reference/projectroundtrip"
	"seme.local/reference/projectsource"
)

type projections map[string]string

func (p projections) String() string { return "path=source" }
func (p projections) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("project_round_trip_publish.projection")
	}
	if _, ok := p[parts[0]]; ok {
		return fmt.Errorf("project_round_trip_publish.projection_duplicate:%s", parts[0])
	}
	p[parts[0]] = parts[1]
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var root, snapshotPath, dest, tracked, ignoredPrefixes, ignoredSuffixes, vendoredPrefixes, generatedHeader string
	var maxFiles int
	var maxFileBytes, maxTotalBytes int64
	projected := projections{}
	flag.StringVar(&root, "root", "", "verified original project root")
	flag.StringVar(&snapshotPath, "snapshot", "", "detached source snapshot JSON")
	flag.StringVar(&dest, "dest", "", "new projected project directory")
	flag.Var(projected, "projected", "project-relative path=plain source file; repeatable")
	flag.StringVar(&tracked, "tracked-extensions", "", "comma-separated tracked extensions")
	flag.StringVar(&ignoredPrefixes, "ignored-prefixes", "", "comma-separated ignored prefixes")
	flag.StringVar(&ignoredSuffixes, "ignored-suffixes", "", "comma-separated ignored suffixes")
	flag.StringVar(&vendoredPrefixes, "vendored-prefixes", "", "comma-separated vendored prefixes")
	flag.StringVar(&generatedHeader, "generated-header", "", "exact generated header")
	flag.IntVar(&maxFiles, "max-files", 1024, "maximum files")
	flag.Int64Var(&maxFileBytes, "max-file-bytes", 2<<20, "maximum file bytes")
	flag.Int64Var(&maxTotalBytes, "max-total-bytes", 32<<20, "maximum total bytes")
	flag.Parse()
	if root == "" || snapshotPath == "" || dest == "" || tracked == "" || generatedHeader == "" || len(projected) == 0 {
		return fmt.Errorf("project_round_trip_publish.missing")
	}
	raw, err := strictRead(snapshotPath)
	if err != nil {
		return err
	}
	var snapshot projectsource.Snapshot
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&snapshot); err != nil {
		return err
	}
	var extra any
	if trailing := decoder.Decode(&extra); trailing != io.EOF {
		return fmt.Errorf("project_round_trip_publish.snapshot_trailing")
	}
	policy := projectsource.Policy{TrackedExtensions: split(tracked), IgnoredPrefixes: split(ignoredPrefixes), IgnoredSuffixes: split(ignoredSuffixes), VendoredPrefixes: split(vendoredPrefixes), GeneratedHeader: []byte(generatedHeader), MaxFiles: maxFiles, MaxFileBytes: maxFileBytes, MaxTotalBytes: maxTotalBytes}
	bundle, err := projectbundle.Capture(root, snapshot, policy)
	if err != nil {
		return err
	}
	sources := map[string][]byte{}
	for path, file := range projected {
		data, e := strictRead(file)
		if e != nil {
			return e
		}
		sources[path] = data
	}
	_, err = projectroundtrip.PublishProjected(dest, snapshot, bundle, sources, policy)
	return err
}
func strictRead(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("project_round_trip_publish.not_regular:%s", path)
	}
	return os.ReadFile(path)
}
func split(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, ",")
}
