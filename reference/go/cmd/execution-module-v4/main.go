// Command execution-module-v4 emits Core Execution Semantics revision 4.
package main

import (
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 4); err != nil {
		panic(err)
	}
}
