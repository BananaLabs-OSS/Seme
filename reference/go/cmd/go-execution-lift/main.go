// Command go-execution-lift performs the first exact Go provider lift into
// language-neutral Core Execution Semantics v1.
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
	modulePath := flag.String("module", "", "Core Execution v1 module.g1")
	function := flag.String("function", "Add", "function to lift")
	out := flag.String("out", "", "output G1 path")
	flag.Parse()
	if *project == "" || *manifestPath == "" || *modulePath == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: go-execution-lift --project DIR --manifest FILE --module FILE --function Add --out FILE")
		os.Exit(64)
	}
	manifest, err := goprovider.ReadManifest(*manifestPath)
	check(err)
	module, err := os.ReadFile(*modulePath)
	check(err)
	g1, _, err := goprovider.LiftAdd(*project, manifest, module, *function)
	check(err)
	check(os.WriteFile(*out, []byte(g1), 0o644))
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-execution-lift:", err)
		os.Exit(65)
	}
}
