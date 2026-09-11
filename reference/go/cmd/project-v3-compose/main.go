// Command project-v3-compose combines independently validated neutral project,
// source-inventory, and package-graph authorities into Project v3.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"seme.local/reference/contractcatalog"
	"seme.local/reference/projectv3emitter"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	var execution, packageV1, packageV2, projectV2, projectV3, project, inventory, graph, out string
	flag.StringVar(&execution, "execution-contract", "", "Execution contract")
	flag.StringVar(&packageV1, "package-v1", "", "Package-v1 contract")
	flag.StringVar(&packageV2, "package-v2", "", "Package-v2 contract")
	flag.StringVar(&projectV2, "project-v2", "", "Project-v2 contract")
	flag.StringVar(&projectV3, "project-v3", "", "Project-v3 contract")
	flag.StringVar(&project, "project", "", "Project-v1 artifact")
	flag.StringVar(&inventory, "inventory", "", "source inventory artifact")
	flag.StringVar(&graph, "package-graph", "", "Package-v2 graph artifact")
	flag.StringVar(&out, "out", "", "new Project-v3 artifact")
	flag.Parse()
	values := []string{execution, packageV1, packageV2, projectV2, projectV3, project, inventory, graph, out}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("project_v3_compose.missing")
		}
	}
	ec, e := read(execution)
	if e != nil {
		return e
	}
	p1, e := read(packageV1)
	if e != nil {
		return e
	}
	p2, e := read(packageV2)
	if e != nil {
		return e
	}
	v2, e := read(projectV2)
	if e != nil {
		return e
	}
	v3, e := read(projectV3)
	if e != nil {
		return e
	}
	projectBytes, e := read(project)
	if e != nil {
		return e
	}
	inventoryBytes, e := read(inventory)
	if e != nil {
		return e
	}
	graphBytes, e := read(graph)
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
	artifact, e := projectv3emitter.Emit(projectv3emitter.Input{Contracts: contractsV3, ProjectV2: contractsV2.Project(), Project: projectBytes, Inventory: inventoryBytes, PackageGraph: graphBytes})
	if e != nil {
		return e
	}
	return publish(out, artifact)
}
func read(path string) ([]byte, error) {
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("project_v3_compose.not_regular:%s", path)
	}
	return os.ReadFile(path)
}
func publish(path string, data []byte) (err error) {
	absolute, e := filepath.Abs(path)
	if e != nil {
		return e
	}
	if _, e = os.Lstat(absolute); e == nil {
		return fmt.Errorf("project_v3_compose.output_exists")
	} else if !os.IsNotExist(e) {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(absolute), ".project-v3-compose-*")
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
