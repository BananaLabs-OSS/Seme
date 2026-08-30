// Command target-plan analyzes the scoped ordinary Go package and emits a
// canonical Target Contract v1 execution plan.
package main

import (
	"flag"
	"fmt"
	"os"

	"seme.local/reference/goprovider"
)

func main() {
	project := flag.String("project", "", "ordinary Go project")
	manifestPath := flag.String("manifest", "", "Provider v1 manifest")
	providerGraph := flag.String("provider-graph", "", "Provider ingestion G1")
	foundationModule := flag.String("foundation-module", "", "Foundation module G1")
	executionModule := flag.String("execution-module", "", "Core Execution module G1")
	packageModule := flag.String("package-module", "", "Package Contract module G1")
	targetModule := flag.String("target-module", "", "Target Contract module G1")
	policy := flag.String("policy", "allow-adapted", "allow-adapted or exact-only")
	out := flag.String("out", "", "output G1")
	flag.Parse()
	paths := []string{*providerGraph, *foundationModule, *executionModule, *packageModule, *targetModule}
	if *project == "" || *manifestPath == "" || *out == "" {
		fatal(fmt.Errorf("missing required argument"))
	}
	manifest, err := goprovider.ReadManifest(*manifestPath)
	fatal(err)
	modules := make([][]byte, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			fatal(fmt.Errorf("missing module/graph argument"))
		}
		data, readErr := os.ReadFile(path)
		fatal(readErr)
		modules = append(modules, data)
	}
	graph, err := goprovider.BuildWasmPulpPlan(*project, manifest, modules, *policy)
	fatal(err)
	fatal(os.WriteFile(*out, []byte(graph), 0o644))
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "target-plan:", err)
		os.Exit(65)
	}
}
