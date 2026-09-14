package main

import (
	"log"
	"os"

	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 86); err != nil {
		log.Fatal(err)
	}
}
