// Command execution-module-v10 emits Core Execution Semantics revision 10.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 10); err != nil {
		panic(err)
	}
}
