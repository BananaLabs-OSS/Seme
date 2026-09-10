package main

import (
	"os"
	"seme.local/reference/orderedtransportmodule"
)

func main() {
	if err := orderedtransportmodule.Emit(os.Stdout); err != nil {
		panic(err)
	}
}
