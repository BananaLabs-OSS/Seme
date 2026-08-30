// Command execution-module-v6 emits Core Execution Semantics revision 6.
package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 6); err != nil {
		panic(err)
	}
}
