package main

import (
	"flag"
	"log"
	"os"
	"seme.local/reference/sourcepresentationmodule"
)

func main() {
	outPath := flag.String("out", "", "optional output path")
	flag.Parse()
	out := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		out = f
	}
	if err := sourcepresentationmodule.Emit(out); err != nil {
		log.Fatal(err)
	}
}
