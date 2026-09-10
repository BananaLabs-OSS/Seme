// project-v4-report is a deterministic, read-only Project v4 evidence adapter.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectdependencyinstance"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv4report"
)

const maxArtifact = 64 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "project-v4-report:", err)
		os.Exit(65)
	}
}
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("project-v4-report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	names := []string{"execution", "package-v1", "package-v2", "dependency-contract", "project-v2-contract", "project-v3-contract", "project-v4-contract", "project", "inventory", "package-graph", "project-v3", "dependency", "composed"}
	paths := map[string]*string{}
	for _, n := range names {
		paths[n] = fs.String(n, "", n+" canonical artifact")
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
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
		return fmt.Errorf("contracts_v2:%w", err)
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(a["execution"], a["package-v2"], a["project-v3-contract"])
	if err != nil {
		return fmt.Errorf("contracts_v3:%w", err)
	}
	v4, err := contractcatalog.ResolveProjectContractSetV4(a["execution"], a["package-v2"], a["dependency-contract"], a["project-v4-contract"])
	if err != nil {
		return fmt.Errorf("contracts_v4:%w", err)
	}
	in := projectdependencyinstance.Inputs{Contracts: v4, ProjectV3: projectgraphinstance.Inputs{Contracts: v3, ProjectV2: v2.Project(), Project: a["project"], Inventory: a["inventory"], PackageGraph: a["package-graph"], Composed: a["project-v3"]}, Dependency: a["dependency"], Composed: a["composed"]}
	report, err := projectv4report.Inspect(in)
	if err != nil {
		return err
	}
	encoded, err := encode(report)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, bytes.NewReader(encoded))
	return err
}

func encode(report projectv4report.Report) ([]byte, error) {
	b, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
func readRegular(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("not_regular")
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
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("changed")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxArtifact+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxArtifact {
		return nil, fmt.Errorf("too_large")
	}
	after, err := os.Lstat(path)
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(opened, after) || after.Size() != int64(len(b)) || after.ModTime() != opened.ModTime() {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
