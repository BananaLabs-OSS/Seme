// Command execution-module-v7 emits Core Execution Semantics revision 7.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 7); err != nil {
		panic(err)
	}
}
