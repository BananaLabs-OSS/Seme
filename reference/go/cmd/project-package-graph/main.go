package main

import (
	"fmt"
	"os"

	"seme.local/reference/projectgraph"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: project-package-graph PROJECT.seme")
		os.Exit(64)
	}
	source, err := os.ReadFile(os.Args[1])
	if err == nil {
		var output []byte
		output, err = projectgraph.InspectJSON(source)
		if err == nil {
			_, err = os.Stdout.Write(output)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "project-package-graph:", err)
		os.Exit(65)
	}
}
