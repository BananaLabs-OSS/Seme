// Command go-upb11-reconcile performs the bounded native half of UPB11: lift
// an ordinary project, bind provider mappings to canonical identities, apply
// one cross-package rename, test offline, re-ingest, and atomically publish.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"seme.local/reference/goprovider"
)

type options struct {
	project, out, module, packagePath, entry string
	executionG1, providerG1                  string
	target, expected, replacement            string
	revision                                 uint64
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "go-upb11-reconcile:", err)
		os.Exit(1)
	}
}

func run(arguments []string, stderr io.Writer) error {
	set := flag.NewFlagSet("go-upb11-reconcile", flag.ContinueOnError)
	set.SetOutput(stderr)
	var o options
	set.StringVar(&o.project, "project", "", "absolute ordinary Go project")
	set.StringVar(&o.out, "out", "", "new atomic reconciliation directory")
	set.StringVar(&o.module, "module", "", "Go module path")
	set.StringVar(&o.packagePath, "package", "", "root package path")
	set.StringVar(&o.entry, "entry", "", "root entry function")
	set.StringVar(&o.executionG1, "execution-g1", "", "Execution contract G1")
	set.StringVar(&o.providerG1, "provider-g1", "", "Provider contract G1")
	set.StringVar(&o.target, "target", "", "qualified declaration name")
	set.StringVar(&o.expected, "expected", "", "expected declaration name")
	set.StringVar(&o.replacement, "replacement", "", "replacement declaration name")
	set.Uint64Var(&o.revision, "revision", 0, "nonzero client revision")
	if err := set.Parse(arguments); err != nil || set.NArg() != 0 {
		return fmt.Errorf("arguments")
	}
	if o.project == "" || o.out == "" || o.module == "" || o.packagePath == "" || o.entry == "" || o.executionG1 == "" || o.providerG1 == "" || o.target == "" || o.expected == "" || o.replacement == "" || o.revision == 0 {
		return fmt.Errorf("options")
	}
	snapshot, err := goprovider.ReadDocumentSnapshot(o.project, o.module, o.packagePath, o.entry, o.revision)
	if err != nil {
		return err
	}
	execution, err := os.ReadFile(o.executionG1)
	if err != nil {
		return err
	}
	session, err := goprovider.NewIncrementalSession(execution)
	if err != nil {
		return err
	}
	lifted := session.Apply(snapshot)
	if !lifted.Accepted || !lifted.Valid || lifted.LastValidRevision != o.revision {
		return fmt.Errorf("lift:%v", lifted.Diagnostics)
	}
	canonicalFunctions := make([]goprovider.SourceIdentity, 0, len(lifted.Sources))
	for _, source := range lifted.Sources {
		if source.Kind == "function" {
			canonicalFunctions = append(canonicalFunctions, source)
		}
	}
	manifest, _, err := goprovider.Ingest(goprovider.IngestOptions{Project: o.project, ModuleG1: o.providerG1, CanonicalSources: canonicalFunctions})
	if err != nil {
		return err
	}
	target := ""
	for _, declaration := range manifest.Declarations {
		if declaration.Qualified == o.target && declaration.Name == o.expected {
			if target != "" {
				return fmt.Errorf("target_ambiguous")
			}
			target = declaration.ID
		}
	}
	if target == "" {
		return fmt.Errorf("target_missing")
	}
	_, err = goprovider.ProjectRenameBundle(o.project, o.out, o.providerG1, manifest, target, o.expected, o.replacement)
	return err
}
