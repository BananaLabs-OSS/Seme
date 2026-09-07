// Command execution-module-v9 emits Core Execution Semantics revision 9.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 9); err != nil {
		panic(err)
	}
}
