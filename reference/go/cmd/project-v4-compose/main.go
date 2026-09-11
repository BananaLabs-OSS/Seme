// Command project-v4-compose binds an authenticated Project-v3 graph and
// Dependency-v1 closure into neutral Project v4.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectgraphinstance"
	"seme.local/reference/projectv4emitter"
)

func main() {
	if e := run(); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
func run() error {
	names := []string{"execution-contract", "package-v1", "package-v2", "dependency-contract", "project-v2", "project-v3-contract", "project-v4-contract", "project-v1", "inventory-v2", "package-graph-v2", "project-v3", "dependency", "out"}
	values := make([]string, len(names))
	for i, name := range names {
		flag.StringVar(&values[i], name, "", name)
	}
	flag.Parse()
	for i, value := range values {
		if value == "" {
			return fmt.Errorf("project_v4_compose.missing:%s", names[i])
		}
	}
	readAt := func(i int) ([]byte, error) { return read(values[i]) }
	ec, e := readAt(0)
	if e != nil {
		return e
	}
	p1, e := readAt(1)
	if e != nil {
		return e
	}
	p2, e := readAt(2)
	if e != nil {
		return e
	}
	dc, e := readAt(3)
	if e != nil {
		return e
	}
	v2, e := readAt(4)
	if e != nil {
		return e
	}
	v3, e := readAt(5)
	if e != nil {
		return e
	}
	v4, e := readAt(6)
	if e != nil {
		return e
	}
	project, e := readAt(7)
	if e != nil {
		return e
	}
	inventory, e := readAt(8)
	if e != nil {
		return e
	}
	graph, e := readAt(9)
	if e != nil {
		return e
	}
	composed, e := readAt(10)
	if e != nil {
		return e
	}
	dependency, e := readAt(11)
	if e != nil {
		return e
	}
	contractsV2, e := contractcatalog.ResolveProjectContractSetV2(ec, p1, v2)
	if e != nil {
		return e
	}
	contractsV3, e := contractcatalog.ResolveProjectContractSetV3(ec, p2, v3)
	if e != nil {
		return e
	}
	contractsV4, e := contractcatalog.ResolveProjectContractSetV4(ec, p2, dc, v4)
	if e != nil {
		return e
	}
	inputs := projectgraphinstance.Inputs{Contracts: contractsV3, ProjectV2: contractsV2.Project(), Project: project, Inventory: inventory, PackageGraph: graph, Composed: composed}
	artifact, e := projectv4emitter.Emit(projectv4emitter.Input{Contracts: contractsV4, ProjectV3: inputs, Dependency: dependency})
	if e != nil {
		return e
	}
	return publish(values[12], artifact)
}
func read(path string) ([]byte, error) {
	i, e := os.Lstat(path)
	if e != nil || !i.Mode().IsRegular() {
		return nil, fmt.Errorf("project_v4_compose.not_regular:%s", path)
	}
	return os.ReadFile(path)
}
func publish(path string, data []byte) (err error) {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(absolute); e == nil {
		return fmt.Errorf("project_v4_compose.output_exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(absolute), ".project-v4-compose-*")
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
