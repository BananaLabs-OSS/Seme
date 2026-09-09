package main

import (
	"log"
	"os"
	"seme.local/reference/executionmodule"
)

func main() {
	if err := executionmodule.Emit(os.Stdout, 35); err != nil {
		log.Fatal(err)
	}
}
