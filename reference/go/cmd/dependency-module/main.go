package main

import (
	"os"
	"seme.local/reference/dependencymodule"
)

func main() {
	if err := dependencymodule.Emit(os.Stdout); err != nil {
		panic(err)
	}
}
