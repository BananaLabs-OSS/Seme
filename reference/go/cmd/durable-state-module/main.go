package main

import (
	"os"
	"seme.local/reference/durablestatemodule"
)

func main() {
	if err := durablestatemodule.Emit(os.Stdout); err != nil {
		panic(err)
	}
}
