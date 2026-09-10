package main

import (
	"os"
	"seme.local/reference/controlledeffectsmodule"
)

func main() {
	if err := controlledeffectsmodule.Emit(os.Stdout); err != nil {
		panic(err)
	}
}
