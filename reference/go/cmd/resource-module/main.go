package main

import (
	"os"
	"seme.local/reference/resourcemodule"
)

func main() {
	if err := resourcemodule.Emit(os.Stdout); err != nil {
		panic(err)
	}
}
