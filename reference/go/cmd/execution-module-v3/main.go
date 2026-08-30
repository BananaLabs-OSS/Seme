// Command execution-module-v3 emits Core Execution Semantics revision 3.
package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 3); err != nil {
		panic(err)
	}
}
