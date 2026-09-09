package main

import (
	"fmt"
	"os"
	"seme.local/reference/projectsnapshot"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(64)
	}
	b, e := os.ReadFile(os.Args[1])
	if e == nil {
		e = projectsnapshot.Validate(b)
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(65)
	}
}
