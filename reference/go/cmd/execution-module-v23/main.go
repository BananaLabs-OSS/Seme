package main

import (
	"fmt"
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 23); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
