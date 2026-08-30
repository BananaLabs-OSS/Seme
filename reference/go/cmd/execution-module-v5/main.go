// Command execution-module-v5 emits Core Execution Semantics revision 5.
package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 5); err != nil {
		panic(err)
	}
}
