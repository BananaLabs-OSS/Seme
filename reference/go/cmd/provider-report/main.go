// Command provider-report emits the canonical Provider Contract v1 projection
// report graph from independently persisted operational evidence.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/goprovider"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: provider-report MODULE_G1 REPORT_JSON REINGESTED_MANIFEST")
		os.Exit(64)
	}
	module, err := os.ReadFile(os.Args[1])
	check(err)
	report, err := goprovider.ReadProjectionReport(os.Args[2])
	check(err)
	manifest, err := goprovider.ReadManifest(os.Args[3])
	check(err)
	g1, err := goprovider.EmitProjectionG1(module, report, manifest)
	check(err)
	fmt.Print(g1)
}
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "provider-report:", err)
		os.Exit(65)
	}
}
