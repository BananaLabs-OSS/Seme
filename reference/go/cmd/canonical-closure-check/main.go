// Command canonical-closure-check validates one bounded closed canonical wire
// envelope in linear time.
package main

import (
	"fmt"
	"os"

	"seme.local/reference/canonicalclosure"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "canonical-closure-check: path")
		os.Exit(1)
	}
	source, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "canonical-closure-check:", err)
		os.Exit(1)
	}
	if err = canonicalclosure.Validate(source); err != nil {
		fmt.Fprintln(os.Stderr, "canonical-closure-check:", err)
		os.Exit(1)
	}
}
