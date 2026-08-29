// Command go-provider is the first Provider Contract v1 proof adapter. Its
// deliberately narrow profile is not a claim of general Go support.
package main

import (
	"flag"
	"fmt"
	"os"

	"seme.local/reference/goprovider"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "ingest":
		ingest(os.Args[2:])
	case "rename":
		rename(os.Args[2:])
	default:
		usage()
	}
}

func ingest(arguments []string) {
	flags := flag.NewFlagSet("ingest", flag.ExitOnError)
	project := flags.String("project", "", "ordinary Go project")
	module := flags.String("module", "", "Provider Contract v1 module.g1")
	out := flags.String("out", "", "ingestion output directory")
	priorPath := flags.String("prior", "", "prior manifest for identity recovery")
	flags.Parse(arguments)
	if *project == "" || *module == "" || *out == "" {
		usage()
	}
	var prior *goprovider.Manifest
	if *priorPath != "" {
		loaded, err := goprovider.ReadManifest(*priorPath)
		check(err)
		prior = &loaded
	}
	manifest, g1, err := goprovider.Ingest(goprovider.IngestOptions{Project: *project, ModuleG1: *module, Prior: prior})
	check(err)
	check(goprovider.WriteIngestion(*out, manifest, g1))
}
func rename(arguments []string) {
	flags := flag.NewFlagSet("rename", flag.ExitOnError)
	project := flags.String("project", "", "ordinary Go project")
	manifestPath := flags.String("manifest", "", "current ingestion manifest")
	target := flags.String("target", "", "semantic declaration identity")
	base := flags.String("base", "", "validated canonical base workspace")
	candidate := flags.String("candidate", "", "validated canonical Patch v1 result")
	report := flags.String("report", "", "projection report path")
	validate := flags.Bool("validate", false, "run go test ./... in an isolated staged project")
	flags.Parse(arguments)
	if *project == "" || *manifestPath == "" || *target == "" || *base == "" || *candidate == "" || *report == "" {
		usage()
	}
	manifest, err := goprovider.ReadManifest(*manifestPath)
	check(err)
	result, err := goprovider.RenameFromGraphs(*project, manifest, *target, *base, *candidate, *validate)
	check(err)
	check(goprovider.WriteProjectionReport(*report, result))
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-provider:", err)
		os.Exit(65)
	}
}
func usage() { fmt.Fprintln(os.Stderr, "usage: go-provider ingest|rename [options]"); os.Exit(64) }
