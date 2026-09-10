// project-v5-report is a deterministic read-only Project v5 evidence adapter.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv5instance"
	"seme.local/reference/projectv5report"
)

const maxArtifact = 64 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "project-v5-report:", err)
		os.Exit(65)
	}
}
func run(args []string, out io.Writer) error {
	f := flag.NewFlagSet("project-v5-report", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	names := []string{"execution", "package-v1", "package-v2", "package-v3-contract", "dependency-contract", "project-v2-contract", "project-v3-contract", "project-v4-contract", "project-v5-contract", "project", "inventory", "package-graph", "project-v3", "dependency", "project-v4", "package-v3", "composed"}
	paths := map[string]*string{}
	for _, n := range names {
		paths[n] = f.String(n, "", n+" canonical artifact")
	}
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected_arguments")
	}
	a := map[string][]byte{}
	for _, n := range names {
		if *paths[n] == "" {
			return fmt.Errorf("missing_%s", n)
		}
		b, err := readRegular(*paths[n])
		if err != nil {
			return fmt.Errorf("%s:%w", n, err)
		}
		a[n] = b
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(a["execution"], a["package-v1"], a["project-v2-contract"])
	if err != nil {
		return err
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(a["execution"], a["package-v2"], a["project-v3-contract"])
	if err != nil {
		return err
	}
	v4, err := contractcatalog.ResolveProjectContractSetV4(a["execution"], a["package-v2"], a["dependency-contract"], a["project-v4-contract"])
	if err != nil {
		return err
	}
	v5, err := contractcatalog.ResolveProjectContractSetV5(a["execution"], a["package-v3-contract"], a["dependency-contract"], a["project-v5-contract"])
	if err != nil {
		return err
	}
	pv3 := projectgraphinstance.Inputs{Contracts: v3, ProjectV2: v2.Project(), Project: a["project"], Inventory: a["inventory"], PackageGraph: a["package-graph"], Composed: a["project-v3"]}
	pv4 := projectdependencyinstance.Inputs{Contracts: v4, ProjectV3: pv3, Dependency: a["dependency"], Composed: a["project-v4"]}
	in := projectv5instance.Inputs{Contracts: v5, ProjectV4: pv4, PackageV2: a["package-graph"], PackageV3: a["package-v3"], Composed: a["composed"]}
	report, err := projectv5report.Inspect(in)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	_, err = io.Copy(out, bytes.NewReader(encoded))
	return err
}
func readRegular(path string) ([]byte, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return nil, fmt.Errorf("absolute_path")
	}
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("not_regular")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return nil, fmt.Errorf("symlink_component")
	}
	if before.Size() > maxArtifact {
		return nil, fmt.Errorf("too_large")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxArtifact+1))
	if err != nil {
		return nil, err
	}
	after, pe := os.Lstat(path)
	if pe != nil || !os.SameFile(opened, after) || before.Size() != opened.Size() || before.Size() != after.Size() || int64(len(b)) != before.Size() || !before.ModTime().Equal(opened.ModTime()) || !before.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
