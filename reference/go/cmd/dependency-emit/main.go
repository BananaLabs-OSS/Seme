// Command dependency-emit publishes a provider-resolved neutral Dependency-v1
// closure after independent model and contract validation.
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
	"seme.local/reference/dependencyemitter"
	"seme.local/reference/dependencyinstance"
	"seme.local/reference/dependencyresolution"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	var closurePath, contractPath, out string
	flag.StringVar(&closurePath, "closure", "", "resolved closure JSON")
	flag.StringVar(&contractPath, "contract", "", "Dependency-v1 contract")
	flag.StringVar(&out, "out", "", "new dependency artifact")
	flag.Parse()
	if closurePath == "" || contractPath == "" || out == "" {
		return fmt.Errorf("dependency_emit.missing")
	}
	raw, e := read(closurePath)
	if e != nil {
		return e
	}
	var closure dependencyresolution.Closure
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e = d.Decode(&closure); e != nil {
		return fmt.Errorf("dependency_emit.closure:%w", e)
	}
	var extra any
	if trailing := d.Decode(&extra); trailing != io.EOF {
		return fmt.Errorf("dependency_emit.trailing")
	}
	if e = dependencyresolution.Validate(closure); e != nil {
		return e
	}
	contractBytes, e := read(contractPath)
	if e != nil {
		return e
	}
	contract, e := contractcatalog.ResolveDependencyContract(contractBytes)
	if e != nil {
		return e
	}
	artifact, e := dependencyemitter.Emit(contract, closure)
	if e != nil {
		return e
	}
	if _, e = dependencyinstance.Validate(contract, artifact); e != nil {
		return e
	}
	return publish(out, artifact)
}
func read(path string) ([]byte, error) {
	i, e := os.Lstat(path)
	if e != nil || !i.Mode().IsRegular() {
		return nil, fmt.Errorf("dependency_emit.not_regular:%s", path)
	}
	return os.ReadFile(path)
}
func publish(path string, data []byte) (err error) {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(absolute); e == nil {
		return fmt.Errorf("dependency_emit.output_exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(absolute), ".dependency-emit-*")
	if e != nil {
		return e
	}
	name := f.Name()
	linked := false
	defer func() {
		if !linked {
			_ = os.Remove(name)
		}
	}()
	if _, e = f.Write(data); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Link(name, absolute); e != nil {
		return e
	}
	linked = true
	return os.Remove(name)
}
