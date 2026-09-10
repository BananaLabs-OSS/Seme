// dependency-report is a deterministic, read-only Dependency v1 evidence adapter.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/dependencyreport"
)

const maxArtifact = 64 << 20

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "dependency-report:", err)
		os.Exit(65)
	}
}
func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("dependency-report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	contractPath := fs.String("contract", "", "Dependency v1 contract")
	instancePath := fs.String("instance", "", "Dependency v1 instance")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected_arguments")
	}
	if *contractPath == "" || *instancePath == "" {
		return fmt.Errorf("missing_artifact")
	}
	contractBytes, err := readRegular(*contractPath)
	if err != nil {
		return fmt.Errorf("contract:%w", err)
	}
	instance, err := readRegular(*instancePath)
	if err != nil {
		return fmt.Errorf("instance:%w", err)
	}
	contract, err := contractcatalog.ResolveDependencyContract(contractBytes)
	if err != nil {
		return err
	}
	report, err := dependencyreport.Inspect(contract, instance)
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
