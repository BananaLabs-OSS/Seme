package main

import (
	"flag"
	"fmt"
	"os"

	"seme.local/reference/goprojector"
)

func main() {
	packageName := flag.String("package", "projected", "Go package name")
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: go-projector [-package name] input.g1 output.go")
		os.Exit(64)
	}
	input, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fail(err)
	}
	output, err := goprojector.Project(input, *packageName)
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(flag.Arg(1), output, 0o644); err != nil {
		fail(err)
	}
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(65) }
