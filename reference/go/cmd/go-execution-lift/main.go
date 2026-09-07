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
	profile := flag.String("profile", "add-v1", "exact lift profile: add-v1, quota-v1, structured-v8, or control-v13")
	packageModule := flag.String("package-module", "", "optional Package Contract module G1")
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
	var g1, functionID string
	if *profile == "add-v1" {
		g1, functionID, err = goprovider.LiftAdd(*project, manifest, module, *function)
	} else if *profile == "quota-v1" {
		g1, functionID, err = goprovider.LiftAdmit(*project, manifest, module, *function)
	} else if *profile == "structured-v8" {
		g1, functionID, err = goprovider.LiftStructuredFunction(*project, manifest, module, *function)
	} else if *profile == "control-v13" {
		g1, functionID, err = goprovider.LiftControlFunction(*project, manifest, module, *function)
	} else {
		err = fmt.Errorf("unknown profile %q", *profile)
	}
	check(err)
	if *packageModule != "" {
		packageBytes, readErr := os.ReadFile(*packageModule)
		check(readErr)
		g1, err = goprovider.AttachPackageContract(g1, packageBytes, manifest, functionID)
		check(err)
	}
	check(os.WriteFile(*out, []byte(g1), 0o644))
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "go-execution-lift:", err)
		os.Exit(65)
	}
}
