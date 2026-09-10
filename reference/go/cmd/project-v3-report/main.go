// project-v3-report is a deterministic, read-only Project v3 evidence adapter.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv3report"
)

const maxArtifact = 64 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "project-v3-report:", err)
		os.Exit(65)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("project-v3-report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	names := []string{"execution", "package-v1", "package-v2", "project-v2-contract", "project-v3-contract", "project", "inventory", "package-graph", "composed"}
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
	artifacts := map[string][]byte{}
	for _, n := range names {
		if *paths[n] == "" {
			return fmt.Errorf("missing_%s", n)
		}
		b, err := readRegular(*paths[n])
		if err != nil {
			return fmt.Errorf("%s:%w", n, err)
		}
		artifacts[n] = b
	}
	v2, err := contractcatalog.ResolveProjectContractSetV2(artifacts["execution"], artifacts["package-v1"], artifacts["project-v2-contract"])
	if err != nil {
		return fmt.Errorf("contracts_v2:%w", err)
	}
	v3, err := contractcatalog.ResolveProjectContractSetV3(artifacts["execution"], artifacts["package-v2"], artifacts["project-v3-contract"])
	if err != nil {
		return fmt.Errorf("contracts_v3:%w", err)
	}
	report, err := projectv3report.Inspect(projectgraphinstance.Inputs{Contracts: v3, ProjectV2: v2.Project(), Project: artifacts["project"], Inventory: artifacts["inventory"], PackageGraph: artifacts["package-graph"], Composed: artifacts["composed"]})
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

func encode(report projectv3report.Report) ([]byte, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
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
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(opened, after) || after.Size() != int64(len(b)) {
		return nil, fmt.Errorf("changed")
	}
	return b, nil
}
