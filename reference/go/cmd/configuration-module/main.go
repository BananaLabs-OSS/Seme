package main

import (
	"log"
	"os"
	"seme.local/reference/configurationmodule"
)

func main() {
	if err := configurationmodule.Emit(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
