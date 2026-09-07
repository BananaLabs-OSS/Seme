// Command execution-module-v11 emits Core Execution Semantics revision 11.
package main

import (
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 11); err != nil {
		panic(err)
	}
}
