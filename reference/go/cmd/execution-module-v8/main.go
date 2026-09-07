// Command execution-module-v8 emits Core Execution Semantics revision 8.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 8); err != nil {
		panic(err)
	}
}
