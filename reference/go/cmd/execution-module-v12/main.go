// Command execution-module-v12 emits Core Execution Semantics revision 12.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 12); err != nil {
		panic(err)
	}
}
