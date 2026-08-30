// Command execution-module-v2 emits frozen Core Execution Semantics revision 2.
package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 2); err != nil {
		panic(err)
	}
}
