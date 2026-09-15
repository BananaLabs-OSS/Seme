package main

import (
	"log"
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 112); err != nil {
		log.Fatal(err)
	}
}
