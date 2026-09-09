// Command project-module emits Project Contract v1.
package main

import (
	"flag"
	"log"
	"os"
	"seme.local/reference/projectmodule"
)

func main() {
	outPath := flag.String("out", "", "optional output path")
	flag.Parse()
	out := os.Stdout
	if *outPath != "" {
		file, err := os.Create(*outPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		out = file
	}
	if err := projectmodule.Emit(out); err != nil {
		log.Fatal(err)
	}
}
